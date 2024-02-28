package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vladislav-kr/gophermart/internal/api/handlers"
	httpserver "github.com/vladislav-kr/gophermart/internal/api/http-server"
	"github.com/vladislav-kr/gophermart/internal/api/router"
	accrualsystem "github.com/vladislav-kr/gophermart/internal/clients/accrual-system"
	"github.com/vladislav-kr/gophermart/internal/logger"
	"github.com/vladislav-kr/gophermart/internal/service"
	passwordgenerator "github.com/vladislav-kr/gophermart/internal/service/password-generator"
	retrieveupdates "github.com/vladislav-kr/gophermart/internal/service/retrieve-updates"
	"github.com/vladislav-kr/gophermart/internal/storage/postgres"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

type HTTP struct {
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type AccrualSystem struct {
	URI           string
	RetryCount    int
	RetryWaitTime time.Duration
	ReadTimeout   time.Duration
}

type Clients struct {
	Accrual AccrualSystem
}

type WorkerUpdateOrdes struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	ReadLimit    uint32
	WorkersLimit uint8
}

type Workers struct {
	UpdateOrders WorkerUpdateOrdes
}

type PostgresStorage struct {
	URI string
}

type Storages struct {
	Postgres PostgresStorage
}

type Option struct {
	HTTP     HTTP
	Clients  Clients
	Storages Storages
	Workers  Workers
}

type App struct {
	closers []io.Closer
	opt     Option
}

func NewApp(opt Option) *App {
	return &App{
		closers: make([]io.Closer, 0, 1),
		opt:     opt,
	}
}

func (a *App) Run(ctx context.Context) error {
	log := logger.Logger().With(slog.String("component", "app"))

	// Контекст прослушивающий сигналы прерывания
	sigCtx, sigCancel := signal.NotifyContext(ctx,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer sigCancel()

	storage, err := postgres.New(ctx, postgres.Config{URI: a.opt.Storages.Postgres.URI})
	if err != nil {
		return err
	}

	a.closers = append(a.closers, storage)

	key, err := rsa.GenerateKey(rand.Reader, 2<<10)
	if err != nil {
		return fmt.Errorf("generate RSA private key: %w", err)
	}

	accrual := accrualsystem.New(
		a.opt.Clients.Accrual.URI,
		accrualsystem.WithRetry(
			a.opt.Clients.Accrual.RetryCount,
			a.opt.Clients.Accrual.RetryWaitTime,
		),
	)
	passGen := passwordgenerator.New(bcrypt.DefaultCost)

	updater := retrieveupdates.New(
		accrual,
		storage,
		ctx.Done(),
		a.opt.Clients.Accrual.ReadTimeout,
		a.opt.Workers.UpdateOrders.ReadTimeout,
		a.opt.Workers.UpdateOrders.WriteTimeout,
		a.opt.Workers.UpdateOrders.ReadLimit,
		a.opt.Workers.UpdateOrders.WorkersLimit,
	)

	go func() {
		for {
			select {
			case err := <-updater.Error():
				log.Error("update worker returned an error", logger.Error(err))
			case <-ctx.Done():
				return
			}
		}
	}()

	srv := &http.Server{
		Addr: a.opt.HTTP.Host,
		Handler: router.NewRouter(
			handlers.NewHandlers(
				service.NewService(passGen, storage, accrual, key),
				storage,
			),
			&key.PublicKey,
		),
		ReadTimeout:  a.opt.HTTP.ReadTimeout,
		WriteTimeout: a.opt.HTTP.WriteTimeout,
		IdleTimeout:  a.opt.HTTP.IdleTimeout,
	}

	httpServer := httpserver.NewHTTPServer(srv)

	// Группа для запуска и остановки сервера по сигналу
	errGr, errGrCtx := errgroup.WithContext(sigCtx)

	errGr.Go(func() error {
		return httpServer.Run()
	})

	errGr.Go(func() error {
		defer func() {
			for _, closer := range a.closers {
				if err := closer.Close(); err != nil {
					logger.Logger().Error(
						"close connection",
						logger.Error(err),
					)
				}
			}
		}()
		<-errGrCtx.Done()

		ctx, cancel := context.WithTimeout(
			context.Background(),
			a.opt.HTTP.ShutdownTimeout,
		)
		defer cancel()

		return httpServer.Stop(ctx)
	})

	return errGr.Wait()

}
