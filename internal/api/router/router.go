package router

import (
	"crypto/rsa"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/vladislav-kr/gofermart-bonus/internal/api/handlers"
	"github.com/vladislav-kr/gofermart-bonus/internal/logger"
)

// NewRouter конфигурирует главный роутер
func NewRouter(h *handlers.Handlers, publicKey *rsa.PublicKey) *chi.Mux {
	log := logger.HTTPLogger()

	auth := jwtauth.New(jwa.RS256.String(), publicKey, nil)

	router := chi.NewRouter()

	router.Group(func(r chi.Router) {
		r.Use(
			middleware.Recoverer,
			middleware.RequestID,
			middleware.RealIP,
			httplog.RequestLogger(log),
		)
		r.Group(func(r chi.Router) {
			// регистрация пользователя;
			r.Method(http.MethodPost, "/api/user/register", handlers.Handler(h.Register))

			// аутентификация пользователя
			r.Method(http.MethodPost, "/api/user/login", handlers.Handler(h.Login))
		})

		r.Group(func(r chi.Router) {
			r.Use(
				middleware.Compress(5),
				jwtauth.Verifier(auth),
				jwtauth.Authenticator(auth),
			)

			//загрузка пользователем номера заказа для расчёта
			r.Method(http.MethodPost, "/api/user/orders", handlers.Handler(h.SaveOrder))

			//получение списка загруженных пользователем номеров заказов,
			//статусов их обработки и информации о начислениях
			r.Method(http.MethodGet, "/api/user/orders", handlers.Handler(h.ListOrdersByUser))

			//получение текущего баланса счёта баллов лояльности пользователя
			r.Method(http.MethodGet, "/api/user/balance", handlers.Handler(h.BalanceByUser))

			//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
			r.Method(http.MethodPost, "/api/user/balance/withdraw", handlers.Handler(h.WithdrawBonuses))

			//получение информации о выводе средств с накопительного счёта пользователем
			r.Method(http.MethodGet, "/api/user/withdrawals", handlers.Handler(h.HistoryWithdrawals))
		})

		//готов принимать запросы
		r.Method(http.MethodGet, "/ready", handlers.Handler(h.Ready))
	})

	//запустился
	router.Get("/live", h.Live)

	return router
}
