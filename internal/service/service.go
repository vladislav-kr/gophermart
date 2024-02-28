package service

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vladislav-kr/gophermart/internal/clients"
	"github.com/vladislav-kr/gophermart/internal/domain/models"
	"github.com/vladislav-kr/gophermart/internal/logger"
	"github.com/vladislav-kr/gophermart/internal/service/jwt"
	"github.com/vladislav-kr/gophermart/internal/storage"
)

//go:generate mockery --name Storage
type Storage interface {
	CreateUser(ctx context.Context, login string, passwordHash []byte) (string, error)
	User(ctx context.Context, login string) (*storage.User, error)
	CreateOrder(ctx context.Context, userID string, order storage.CreateOrder) error
	Orders(ctx context.Context, userID string) ([]storage.Order, error)
	UserBalance(ctx context.Context, userID string) (*storage.Balance, error)
	Withdrawals(ctx context.Context, userID string) ([]storage.WithdrawalsBonuses, error)
	Withdraw(ctx context.Context, userID string, withdraw storage.WithdrawBonuses) error
}

//go:generate mockery --name Accrual
type Accrual interface {
	Order(ctx context.Context, orderID string) (*clients.OrderAccrual, time.Duration, error)
}

//go:generate mockery --name PasswordGenerator
type PasswordGenerator interface {
	CompareHashAndPassword(hashedPassword, password []byte) error
	GenerateFromPassword(password []byte) ([]byte, error)
}

type service struct {
	generator  PasswordGenerator
	storage    Storage
	accrual    Accrual
	privateKey *rsa.PrivateKey
	log        *slog.Logger
}

func NewService(g PasswordGenerator, s Storage, a Accrual, privateKey *rsa.PrivateKey) *service {
	return &service{
		generator:  g,
		storage:    s,
		accrual:    a,
		privateKey: privateKey,
		log:        logger.Logger().With(slog.String("component", "service")),
	}
}

func (s *service) Login(ctx context.Context, cred models.Credentials) (string, error) {
	if err := cred.Validate(); err != nil {
		return "", models.ErrIncorrectCredentials
	}

	user, err := s.storage.User(ctx, cred.Login)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNoRecordsFound):
			return "", models.ErrIncorrectCredentials
		default:
			return "", fmt.Errorf("storage user %v: %w", err, models.ErrInternal)
		}
	}

	if err := s.generator.CompareHashAndPassword(
		user.Password,
		[]byte(cred.Password),
	); err != nil {
		switch {
		case errors.Is(err, models.ErrMismatchedHashAndPassword):
			return "", models.ErrIncorrectCredentials
		default:
			return "", fmt.Errorf("compare hash and password %v: %w", err, models.ErrInternal)
		}
	}

	token, err := jwt.NewToken(user.UserID, time.Minute*15, s.privateKey)
	if err != nil {
		return "", fmt.Errorf("token generation %v: %w", err, models.ErrInternal)
	}

	return token, nil
}

func (s *service) Register(ctx context.Context, cred models.Credentials) (string, error) {

	if err := cred.Validate(); err != nil {
		return "", models.ErrIncorrectCredentials
	}

	passHash, err := s.generator.GenerateFromPassword([]byte(cred.Password))
	if err != nil {
		return "", fmt.Errorf("generate hash password %v: %w", err, models.ErrInternal)
	}

	userUUID, err := s.storage.CreateUser(ctx, cred.Login, passHash)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUniqueViolation):
			return "", models.ErrLoginAlreadyExists
		default:
			return "", fmt.Errorf("create user %v: %w", err, models.ErrInternal)
		}
	}

	token, err := jwt.NewToken(userUUID, time.Minute*15, s.privateKey)
	if err != nil {
		return "", fmt.Errorf("token generation %v: %w", err, models.ErrInternal)
	}

	return token, nil

}

func (s *service) Order(ctx context.Context, orderID models.OrderID, userID models.UserID) error {
	if !userID.Validate() {
		return models.ErrUserIDMandatory
	}

	if !orderID.Validate() {
		return models.ErrIncorrectOrderNumber
	}

	// получим закал из системы расчетов бонусов
	accrualOrder, _, err := s.accrual.Order(ctx, string(orderID))
	if err != nil {
		accrualOrder = &clients.OrderAccrual{
			Order:  string(orderID),
			Status: models.StatusNew,
		}
	}

	if accrualOrder.Status == "REGISTERED" {
		accrualOrder.Status = models.StatusNew
	}

	// создаем заказ в хранилище
	if err := s.storage.CreateOrder(ctx, string(userID),
		storage.CreateOrder{
			OrderID: accrualOrder.Order,
			Status:  accrualOrder.Status,
			Accrual: accrualOrder.Accural,
		}); err != nil {
		switch {
		case errors.Is(err, storage.ErrAlreadyUploadedUser):
			return fmt.Errorf("create order %v: %w", err, models.ErrAlreadyUploadedUser)
		case errors.Is(err, storage.ErrAlreadyUploadedAnotherUser):
			return fmt.Errorf("create order %v: %w", err, models.ErrAlreadyUploadedAnotherUser)
		}

		return fmt.Errorf("create order %v: %w", err, models.ErrInternal)
	}

	return nil
}

func (s *service) OrdersByUserID(ctx context.Context, userID models.UserID) ([]models.Order, error) {
	if !userID.Validate() {
		return nil, models.ErrUserIDMandatory
	}

	dbOrders, err := s.storage.Orders(ctx, string(userID))
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNoRecordsFound):
			return nil, models.ErrNoRecordsFound
		default:
			return nil, fmt.Errorf("orders %v: %w", err, models.ErrInternal)
		}
	}

	orders := make([]models.Order, 0, len(dbOrders))
	for _, order := range dbOrders {
		orders = append(orders, models.Order{
			OrderID:    models.OrderID(order.OrderID),
			Status:     order.Status,
			UploadedAt: order.UploadedAt,
			Accrual:    order.Accrual,
		})
	}

	return orders, nil
}

func (s *service) UserBalance(ctx context.Context, userID models.UserID) (*models.Balance, error) {
	if !userID.Validate() {
		return nil, models.ErrUserIDMandatory
	}

	balance, err := s.storage.UserBalance(ctx, string(userID))
	if err != nil {
		return nil, fmt.Errorf("user balance %v: %w", err, models.ErrInternal)
	}

	return &models.Balance{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}, nil
}

func (s *service) WithdrawalsByUserID(ctx context.Context, userID models.UserID) ([]models.WithdrawalsBonuses, error) {
	if !userID.Validate() {
		return nil, models.ErrUserIDMandatory
	}

	dbWithdrawals, err := s.storage.Withdrawals(ctx, string(userID))
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrNoRecordsFound):
			return nil, models.ErrNoRecordsFound
		default:
			return nil, fmt.Errorf("withdrawals %v: %w", err, models.ErrInternal)
		}
	}
	withdrawals := make([]models.WithdrawalsBonuses, 0, len(dbWithdrawals))

	for _, w := range dbWithdrawals {
		withdrawals = append(withdrawals, models.WithdrawalsBonuses{
			Order:       models.OrderID(w.Order),
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt,
		})
	}

	return withdrawals, nil
}

func (s *service) Withdraw(ctx context.Context, userID models.UserID, withdraw models.WithdrawBonuses) error {
	if !userID.Validate() {
		return models.ErrUserIDMandatory
	}

	if !withdraw.Order.Validate() {
		return models.ErrIncorrectOrderNumber
	}

	if err := s.storage.Withdraw(ctx, string(userID), storage.WithdrawBonuses{
		Order: string(withdraw.Order),
		Sum:   withdraw.Sum,
	}); err != nil {
		switch {
		case errors.Is(err, storage.ErrConstraints):
			return models.ErrInsufficientFunds
		default:
			return fmt.Errorf("withdraw %v: %w", err, models.ErrInternal)
		}
	}

	return nil

}
