package storage

import (
	"context"
	"errors"
	"io"
)

var (
	ErrUniqueViolation = errors.New("uniqueness is violated")
	ErrNoRecordsFound  = errors.New("no records found")
	ErrConstraints     = errors.New("constraints error")
	ErrInternal        = errors.New("internal")

	ErrAlreadyUploadedUser        = errors.New("already uploaded by user")
	ErrAlreadyUploadedAnotherUser = errors.New("already uploaded by another user")
)

type Storage interface {
	CreateUser(ctx context.Context, login string, passwordHash []byte) (string, error)
	User(ctx context.Context, login string) (*User, error)
	CreateOrder(ctx context.Context, userID string, order CreateOrder) error
	Orders(ctx context.Context, userID string) ([]Order, error)
	UserBalance(ctx context.Context, userID string) (*Balance, error)
	Withdrawals(ctx context.Context, userID string) ([]WithdrawalsBonuses, error)
	Withdraw(ctx context.Context, userID string, withdraw WithdrawBonuses) error
	OrdersForUpdate(ctx context.Context, limit uint32) ([]UpdateOrderID, error)
	BatchUpdateOrder(ctx context.Context, orders []UpdateOrder) error
	Ping(ctx context.Context) error
	io.Closer
}
