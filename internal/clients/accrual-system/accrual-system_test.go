package accrualsystem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/vladislav-kr/gophermart/internal/clients"
)

func TestWithOpton(t *testing.T) {

	opt := apply(WithRetry(1, time.Second))
	assert.Equal(t, opt, &accrualSystem{
		retryCount: 1,
		retryWait:  time.Second,
	})
}

func Test_accrualSystem_Order(t *testing.T) {
	type args struct {
		orderID string
		status  int
	}
	tests := []struct {
		name      string
		args      args
		wantOrder *clients.OrderAccrual
		wantDelay time.Duration
		wantErr   error
	}{
		{
			name: "заказ успешно получен",
			args: args{
				orderID: "order_1",
				status:  http.StatusOK,
			},
			wantOrder: &clients.OrderAccrual{
				Order:   "order_1",
				Status:  "PROCESSED",
				Accural: 500,
			},
		},
		{
			name: "заказ не зарегистрирован в системе расчёта",
			args: args{
				orderID: "order_2",
				status:  http.StatusNoContent,
			},
			wantErr: clients.ErrNotRegistered,
		},
		{
			name: "слишком много запросов",
			args: args{
				orderID: "order_3",
				status:  http.StatusTooManyRequests,
			},
			wantDelay: time.Second * 60,
			wantErr:   clients.ErrManyRequests,
		},
		{
			name: "система бонусов недоступна",
			args: args{
				orderID: "order_4",
				status:  http.StatusInternalServerError,
			},
			wantErr: clients.ErrInternalError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := chi.NewRouter()
			router.Get("/api/orders/{id}",
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.wantOrder != nil {
						render.JSON(w, r, tt.wantOrder)
					}

					if tt.args.status == http.StatusTooManyRequests {
						w.Header().Set("Retry-After", "60")
					}

					w.WriteHeader(tt.args.status)
					w.WriteHeader(tt.args.status)

				}))

			ts := httptest.NewServer(router)
			defer ts.Close()

			client := New(
				ts.URL,
				WithRetry(2, time.Second),
			)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			order, delay, err := client.Order(ctx, tt.args.orderID)

			if tt.wantOrder != nil {
				assert.Equal(t, order, tt.wantOrder)
				assert.Equal(t, delay, time.Duration(0))
				assert.NoError(t, err)
			}

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
			}

			if tt.wantDelay > 0 {
				assert.Equal(t, delay, tt.wantDelay)
			}

		})
	}
}
