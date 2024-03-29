// Code generated by mockery v2.37.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	storage "github.com/vladislav-kr/gophermart/internal/storage"
)

// Storage is an autogenerated mock type for the Storage type
type Storage struct {
	mock.Mock
}

// CreateOrder provides a mock function with given fields: ctx, userID, order
func (_m *Storage) CreateOrder(ctx context.Context, userID string, order storage.CreateOrder) error {
	ret := _m.Called(ctx, userID, order)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, storage.CreateOrder) error); ok {
		r0 = rf(ctx, userID, order)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateUser provides a mock function with given fields: ctx, login, passwordHash
func (_m *Storage) CreateUser(ctx context.Context, login string, passwordHash []byte) (string, error) {
	ret := _m.Called(ctx, login, passwordHash)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte) (string, error)); ok {
		return rf(ctx, login, passwordHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte) string); ok {
		r0 = rf(ctx, login, passwordHash)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, []byte) error); ok {
		r1 = rf(ctx, login, passwordHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Orders provides a mock function with given fields: ctx, userID
func (_m *Storage) Orders(ctx context.Context, userID string) ([]storage.Order, error) {
	ret := _m.Called(ctx, userID)

	var r0 []storage.Order
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]storage.Order, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []storage.Order); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]storage.Order)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// User provides a mock function with given fields: ctx, login
func (_m *Storage) User(ctx context.Context, login string) (*storage.User, error) {
	ret := _m.Called(ctx, login)

	var r0 *storage.User
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*storage.User, error)); ok {
		return rf(ctx, login)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *storage.User); ok {
		r0 = rf(ctx, login)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*storage.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, login)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UserBalance provides a mock function with given fields: ctx, userID
func (_m *Storage) UserBalance(ctx context.Context, userID string) (*storage.Balance, error) {
	ret := _m.Called(ctx, userID)

	var r0 *storage.Balance
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*storage.Balance, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *storage.Balance); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*storage.Balance)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Withdraw provides a mock function with given fields: ctx, userID, withdraw
func (_m *Storage) Withdraw(ctx context.Context, userID string, withdraw storage.WithdrawBonuses) error {
	ret := _m.Called(ctx, userID, withdraw)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, storage.WithdrawBonuses) error); ok {
		r0 = rf(ctx, userID, withdraw)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Withdrawals provides a mock function with given fields: ctx, userID
func (_m *Storage) Withdrawals(ctx context.Context, userID string) ([]storage.WithdrawalsBonuses, error) {
	ret := _m.Called(ctx, userID)

	var r0 []storage.WithdrawalsBonuses
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]storage.WithdrawalsBonuses, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []storage.WithdrawalsBonuses); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]storage.WithdrawalsBonuses)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewStorage creates a new instance of Storage. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStorage(t interface {
	mock.TestingT
	Cleanup(func())
}) *Storage {
	mock := &Storage{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
