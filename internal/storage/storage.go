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
)

type Storage interface {
	CreateUser(ctx context.Context, login string, passwordHash []byte) (string, error)
	User(ctx context.Context, login string) (*User, error)
	OrderByNumber(ctx context.Context, orderID string) (*Order, error)
	CreateOrder(ctx context.Context, userID string, order CreateOrder) error
	Orders(ctx context.Context, userID string) ([]Order, error)
	UserBalance(ctx context.Context, userID string) (*Balance, error)
	Withdrawals(ctx context.Context, userID string) ([]WithdrawalsBonuses, error)
	Withdraw(ctx context.Context, userID string, withdraw WithdrawBonuses) error
	io.Closer
}
