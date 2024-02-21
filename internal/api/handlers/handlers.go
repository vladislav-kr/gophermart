package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/render"

	"github.com/vladislav-kr/gofermart-bonus/internal/domain/models"
	"github.com/vladislav-kr/gofermart-bonus/internal/domain/response"
	"github.com/vladislav-kr/gofermart-bonus/internal/logger"
	"github.com/vladislav-kr/gofermart-bonus/internal/service/jwt"
)

var (
	ErrUserIDNotFound = errors.New("uesrID not found")
)

type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		httplog.LogEntrySetField(
			r.Context(),
			"handler_error",
			slog.AnyValue(err),
		)
	}
}

//go:generate mockery --name service --exported
type service interface {
	Login(ctx context.Context, cred models.Credentials) (string, error)
	Register(ctx context.Context, cred models.Credentials) (string, error)
	Order(ctx context.Context, orderID models.OrderID, userID models.UserID) error
	OrdersByUserID(ctx context.Context, userID models.UserID) ([]models.Order, error)
	UserBalance(ctx context.Context, userID models.UserID) (*models.Balance, error)
	WithdrawalsByUserID(ctx context.Context, userID models.UserID) ([]models.WithdrawalsBonuses, error)
	Withdraw(ctx context.Context, userID models.UserID, withdraw models.WithdrawBonuses) error
}

//go:generate mockery --name pinger --exported
type pinger interface {
	Ping(ctx context.Context) error
}

type Handlers struct {
	log     *slog.Logger
	service service
	pinger  pinger
}

func NewHandlers(s service, p pinger) *Handlers {
	return &Handlers{
		log: logger.Logger().With(
			slog.String("comopnetn", "handlers"),
		),
		service: s,
		pinger:  p,
	}
}

// аутентификация пользователя
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) error {
	cred := models.Credentials{}

	if err := render.DecodeJSON(r.Body, &cred); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("неверный формат запроса"))
		return fmt.Errorf("decoding the request body into JSON: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()

	token, err := h.service.Login(ctx, cred)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrIncorrectCredentials):
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("неверная пара логин/пароль"))
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
		}
		return fmt.Errorf("register user: %w", err)
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response.OK())

	return nil
}

// регистрация пользователя
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) error {
	cred := models.Credentials{}

	if err := render.DecodeJSON(r.Body, &cred); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("неверный формат запроса"))
		return fmt.Errorf("decoding the request body into JSON: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()

	token, err := h.service.Register(ctx, cred)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrLoginAlreadyExists):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, response.Error("логин уже занят"))
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
		}
		return fmt.Errorf("register user: %w", err)
	}

	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response.OK())
	return nil
}

// загрузка пользователем номера заказа для расчёта
func (h *Handlers) SaveOrder(w http.ResponseWriter, r *http.Request) error {
	userID, _:= userIDFromContext(r.Context())

	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil || len(data) == 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("неверный формат запроса"))
		return fmt.Errorf("read all body: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*8)
	defer cancel()

	if err := h.service.Order(ctx, models.OrderID(data), models.UserID(userID)); err != nil {
		switch {
		case errors.Is(err, models.ErrIncorrectOrderNumber):
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, response.Error("неверный формат номера заказа"))

		case errors.Is(err, models.ErrAlreadyUploadedUser):
			render.Status(r, http.StatusOK)
			render.JSON(w, r, response.Error("номер заказа уже был загружен этим пользователем"))

		case errors.Is(err, models.ErrAlreadyUploadedAnotherUser):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, response.Error("номер заказа уже был загружен другим пользователем"))

		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
		}
		return fmt.Errorf("save order: %w", err)
	}

	// новый номер заказа принят в обработку
	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, response.OK())

	return nil
}

// получение списка загруженных пользователем номеров заказов,
// статусов их обработки и информации о начислениях
func (h *Handlers) ListOrdersByUser(w http.ResponseWriter, r *http.Request) error {
	userID, _:= userIDFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()
	orders, err := h.service.OrdersByUserID(ctx, models.UserID(userID))
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecordsFound):
			render.Status(r, http.StatusNoContent)
			render.JSON(w, r, response.OK())
			return nil
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
			return fmt.Errorf("list orders by user: %w", err)
		}
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, orders)
	return nil
}

// получение текущего баланса счёта баллов лояльности пользователя
func (h *Handlers) BalanceByUser(w http.ResponseWriter, r *http.Request) error {
	userID, _:= userIDFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()
	balance, err := h.service.UserBalance(ctx, models.UserID(userID))
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
		return fmt.Errorf("user balance: %w", err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, balance)

	return nil
}

// запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
func (h *Handlers) WithdrawBonuses(w http.ResponseWriter, r *http.Request) error {
	userID, _:= userIDFromContext(r.Context())

	withdraw := models.WithdrawBonuses{}

	err := render.DecodeJSON(r.Body, &withdraw)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
		return fmt.Errorf("decode JSON: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()

	if err := h.service.Withdraw(ctx, models.UserID(userID), withdraw); err != nil {
		switch {
		case errors.Is(err, models.ErrInsufficientFunds):
			render.Status(r, http.StatusPaymentRequired)
			render.JSON(w, r, response.Error("на счету недостаточно средств"))
		case errors.Is(err, models.ErrIncorrectOrderNumber):
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, response.Error("неверный номер заказа"))
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
		}
		return fmt.Errorf("withdraw: %w", err)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response.OK())
	return nil
}

// получение информации о выводе средств с накопительного счёта пользователем
func (h *Handlers) HistoryWithdrawals(w http.ResponseWriter, r *http.Request) error {
	userID, _:= userIDFromContext(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()

	withdrawals, err := h.service.WithdrawalsByUserID(ctx, models.UserID(userID))
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNoRecordsFound):
			render.Status(r, http.StatusNoContent)
			render.JSON(w, r, response.Error("нет ни одного списания"))
			return nil
		default:
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("внутренняя ошибка сервера"))
			return fmt.Errorf("withdrawals: %w", err)
		}
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, withdrawals)
	return nil
}

// сервер запустился
func (h *Handlers) Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// сервер готов принимать запросы
func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second*4)
	defer cancel()
	if err := h.pinger.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("ping db: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

// userID из токена JWT
func userIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := jwt.ClaimJWTFromContext[string](ctx, jwt.UserID)
	if !ok {
		return "", false
	}
	return userID, true
}
