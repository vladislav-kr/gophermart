package retrieveupdates

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vladislav-kr/gofermart-bonus/internal/clients"
	"github.com/vladislav-kr/gofermart-bonus/internal/storage"
)

// locker - блокиратор с ожиданием и уведомлением n-воркеров
type locker struct {
	// чек простоя воркеров
	hasWaiting bool
	// блокировка установки hasWaiting
	waitMutex sync.RWMutex
	// уведомление "необходимо ждать"
	// создается в lock
	waitCh chan struct{}
}

func (l *locker) lock(d time.Duration) {
	l.waitMutex.Lock()
	defer l.waitMutex.Unlock()
	if !l.hasWaiting {
		l.hasWaiting = true
		l.waitCh = make(chan struct{})
		go func() {
			time.Sleep(d)
			close(l.waitCh)
			l.waitMutex.Lock()
			defer l.waitMutex.Unlock()
			l.hasWaiting = false
		}()
	}
}

func (l *locker) wait() {
	l.waitMutex.RLock()
	if l.hasWaiting {
		<-l.waitCh
	}
	l.waitMutex.RUnlock()
}

//go:generate mockery --name Updater
type Updater interface {
	OrdersForUpdate(ctx context.Context, limit uint32) ([]storage.UpdateOrderID, error)
	BatchUpdateOrder(ctx context.Context, orders []storage.UpdateOrder) error
}

//go:generate mockery --name Accrual
type Accrual interface {
	Order(ctx context.Context, orderID string) (*clients.OrderAccrual, time.Duration, error)
}

type retrieveUpdates struct {
	// сигнал внешней остановки
	done <-chan struct{}
	// сигнал остановки воркеров
	exit chan struct{}

	// клиент, для чтения обновленного заказа
	accrual Accrual
	// таймаут получения новых данных заказа
	accrualReadTimeout  time.Duration
	updaterReadTimeout  time.Duration
	updaterWriteTimeout time.Duration

	// читает и обновляет заказ
	update Updater

	// лимит чтения 1 пачки заказов
	readingLimit uint32
	// кол-во одновременно работающих воркеров
	numWorkers uint8
	// канал-приемник заказов для обновления
	orderIn chan storage.UpdateOrderID
	// канал с готовыми данными для обновления
	orderOut chan storage.UpdateOrder

	errCh chan error

	locker *locker
}

func New(a Accrual, u Updater,
	done <-chan struct{},
	accrualReadTimeout time.Duration,
	updaterReadTimeout time.Duration,
	updaterWriteTimeout time.Duration,
	limit uint32,
	numWorkers uint8,
) *retrieveUpdates {
	r := &retrieveUpdates{
		accrual:             a,
		accrualReadTimeout:  accrualReadTimeout,
		update:              u,
		updaterReadTimeout:  updaterReadTimeout,
		updaterWriteTimeout: updaterWriteTimeout,
		done:                done,
		exit:                make(chan struct{}),
		readingLimit:        limit,
		numWorkers:          numWorkers,
		orderIn:             make(chan storage.UpdateOrderID, limit),
		errCh:               make(chan error),
		locker:              &locker{},
	}

	go r.reader()
	go r.updater()

	r.orderOut = r.fanIn(r.fanOut()...)

	go func() {
		<-r.done
		// сигнал для мягкой остановки,
		// сохраняем полученные заказы
		close(r.exit)
	}()

	return r
}

func (r *retrieveUpdates) addErr(err error) {
	if err != nil {
		go func() {
			r.errCh <- err
		}()
	}
}

func (r *retrieveUpdates) Error() <-chan error {
	return r.errCh
}

func (r *retrieveUpdates) updatedOrder(orderID string) (*clients.OrderAccrual, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), r.accrualReadTimeout)
	defer cancel()

	ord, delay, err := r.accrual.Order(ctx, orderID)
	if err != nil {
		switch {
		case errors.Is(err, clients.ErrManyRequests):
			r.locker.lock(delay)
		default:
			r.addErr(fmt.Errorf("read accural order: %w", err))
		}
		return nil, false
	}

	//еще не рассчитан
	if ord.Status == "PROCESSING" || ord.Status == "REGISTERED" {
		return nil, false
	}
	return ord, true
}

// обновление заказов пачками
func (r *retrieveUpdates) updater() {
	orders := make([]storage.UpdateOrder, 0)
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	update := func() {
		if len(orders) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), r.updaterWriteTimeout)
			defer cancel()
			if err := r.update.BatchUpdateOrder(ctx, orders); err != nil {
				r.addErr(fmt.Errorf("batch update order: %w", err))
			}
		}
	}

	for {
		select {
		case order, ok := <-r.orderOut:
			if ok {
				orders = append(orders, order)
			}

			if len(orders) > 10 {
				update()
				orders = orders[:0]
			}

			if !ok {
				update()
				//r.orderIn уже закрыт, r.orderOut дочитали, значит ошибок больше не будет
				close(r.errCh)
				return
			}
		case <-ticker.C:
			if len(orders) == 0 {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), r.updaterWriteTimeout)
			if err := r.update.BatchUpdateOrder(ctx, orders); err != nil {
				r.addErr(fmt.Errorf("batch update order: %w", err))
			}
			cancel()
			orders = orders[:0]

		}
	}
}

// чтение заказов для обвновления
func (r *retrieveUpdates) reader() {
	wg := sync.WaitGroup{}
	ticker := time.NewTicker(time.Second * 5)

	defer func() {
		// дождаться завершения заказов в отправке
		wg.Wait()
		close(r.orderIn)
	}()
	defer ticker.Stop()

	for {
		select {
		case <-r.exit:
			return
		case _, ok := <-ticker.C:
			if !ok {
				return
			}

			r.locker.wait()

			ctx, cancel := context.WithTimeout(context.Background(), r.updaterReadTimeout)
			orders, err := r.update.OrdersForUpdate(ctx, r.readingLimit)
			cancel()
			if err != nil {
				switch {
				case errors.Is(err, storage.ErrNoRecordsFound):
					continue
				default:
					r.addErr(fmt.Errorf("read orders for update: %w", err))
				}
			}

			// отправить в приемник новые заказы
			for _, order := range orders {
				order := order
				wg.Add(1)
				go func() {
					defer wg.Done()
					r.orderIn <- order
				}()
			}
		}
	}

}

func (r *retrieveUpdates) worker() chan storage.UpdateOrder {
	result := make(chan storage.UpdateOrder)

	go func() {
		defer close(result)
		for order := range r.orderIn {
			r.locker.wait()
			ord, ok := r.updatedOrder(order.OrderID)
			if !ok {
				continue
			}
			result <- storage.UpdateOrder{
				UserID:  order.UserID,
				OrderID: ord.Order,
				Status:  ord.Status,
				Accrual: ord.Accural,
			}

		}
	}()

	return result

}

func (r *retrieveUpdates) fanOut() []chan storage.UpdateOrder {
	channels := make([]chan storage.UpdateOrder, r.numWorkers)

	for i := uint8(0); i < r.numWorkers; i++ {
		channels[i] = r.worker()
	}

	return channels
}

func (r *retrieveUpdates) fanIn(
	results ...chan storage.UpdateOrder,
) chan storage.UpdateOrder {
	final := make(chan storage.UpdateOrder)
	var wg sync.WaitGroup

	for _, ch := range results {
		orders := ch
		wg.Add(1)
		go func() {
			defer wg.Done()
			for order := range orders {
				select {
				case <-r.done:
					return
				case final <- order:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(final)
	}()

	return final
}
