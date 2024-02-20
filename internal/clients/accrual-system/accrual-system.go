package accrualsystem

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/vladislav-kr/gofermart-bonus/internal/clients"
)

type accrualSystem struct {
	client     *resty.Client
	retryCount int
	retryWait  time.Duration
}

type Option func(*accrualSystem)

func WithRetry(
	retryCount int,
	retryWaitTime time.Duration,
) Option {
	return func(a *accrualSystem) {
		a.retryCount = retryCount
		a.retryWait = retryWaitTime
	}
}

func New(url string, opts ...Option) *accrualSystem {
	accural := apply(opts...)
	accural.client = resty.New().SetBaseURL(url)

	if accural.retryCount > 0 {
		accural.client.
			SetRetryCount(accural.retryCount).
			SetRetryWaitTime(accural.retryWait).
			AddRetryCondition(
				func(r *resty.Response, err error) bool {
					return err == nil &&
						r.StatusCode() != http.StatusTooManyRequests &&
						r.StatusCode() != http.StatusInternalServerError
				},
			)
	}

	return accural
}
func (a *accrualSystem) Order(ctx context.Context, orderID string) (*clients.OrderAccrual, time.Duration, error) {
	orderAccrual := &clients.OrderAccrual{}

	resp, err := a.client.R().
		SetContext(ctx).
		SetResult(orderAccrual).
		Get(fmt.Sprintf("/api/orders/%s", orderID))

	if err != nil {
		return nil, 0, fmt.Errorf("order %v: %w", err, clients.ErrInternalError)
	}

	if resp.StatusCode() == http.StatusOK {
		return orderAccrual, 0, nil
	}

	if resp.StatusCode() == http.StatusNoContent {
		return nil, 0, fmt.Errorf("order %s: %w", orderID, clients.ErrNotRegistered)
	}

	if resp.StatusCode() == http.StatusTooManyRequests {
		delay, err := strconv.Atoi(resp.Header().Get("Retry-After"))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid response header 'Retry-After': %w", err)
		}

		return nil, time.Duration(delay) * time.Second, clients.ErrManyRequests
	}

	return nil, 0, clients.ErrInternalError
}

func apply(opts ...Option) *accrualSystem {
	accrual := &accrualSystem{}
	for _, fn := range opts {
		fn(accrual)
	}
	return accrual
}
