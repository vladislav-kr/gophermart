// Code generated by mockery v2.37.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	models "github.com/vladislav-kr/gofermart-bonus/internal/domain/models"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// Login provides a mock function with given fields: ctx, cred
func (_m *Service) Login(ctx context.Context, cred models.Credentials) (string, error) {
	ret := _m.Called(ctx, cred)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, models.Credentials) (string, error)); ok {
		return rf(ctx, cred)
	}
	if rf, ok := ret.Get(0).(func(context.Context, models.Credentials) string); ok {
		r0 = rf(ctx, cred)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, models.Credentials) error); ok {
		r1 = rf(ctx, cred)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Order provides a mock function with given fields: ctx, orderID, userID
func (_m *Service) Order(ctx context.Context, orderID models.OrderID, userID models.UserID) error {
	ret := _m.Called(ctx, orderID, userID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, models.OrderID, models.UserID) error); ok {
		r0 = rf(ctx, orderID, userID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// OrdersByUserID provides a mock function with given fields: ctx, userID
func (_m *Service) OrdersByUserID(ctx context.Context, userID models.UserID) ([]models.Order, error) {
	ret := _m.Called(ctx, userID)

	var r0 []models.Order
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID) ([]models.Order, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID) []models.Order); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Order)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, models.UserID) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Register provides a mock function with given fields: ctx, cred
func (_m *Service) Register(ctx context.Context, cred models.Credentials) (string, error) {
	ret := _m.Called(ctx, cred)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, models.Credentials) (string, error)); ok {
		return rf(ctx, cred)
	}
	if rf, ok := ret.Get(0).(func(context.Context, models.Credentials) string); ok {
		r0 = rf(ctx, cred)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, models.Credentials) error); ok {
		r1 = rf(ctx, cred)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UserBalance provides a mock function with given fields: ctx, userID
func (_m *Service) UserBalance(ctx context.Context, userID models.UserID) (*models.Balance, error) {
	ret := _m.Called(ctx, userID)

	var r0 *models.Balance
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID) (*models.Balance, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID) *models.Balance); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Balance)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, models.UserID) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Withdraw provides a mock function with given fields: ctx, userID, withdraw
func (_m *Service) Withdraw(ctx context.Context, userID models.UserID, withdraw models.WithdrawBonuses) error {
	ret := _m.Called(ctx, userID, withdraw)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID, models.WithdrawBonuses) error); ok {
		r0 = rf(ctx, userID, withdraw)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WithdrawalsByUserID provides a mock function with given fields: ctx, userID
func (_m *Service) WithdrawalsByUserID(ctx context.Context, userID models.UserID) ([]models.WithdrawalsBonuses, error) {
	ret := _m.Called(ctx, userID)

	var r0 []models.WithdrawalsBonuses
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID) ([]models.WithdrawalsBonuses, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, models.UserID) []models.WithdrawalsBonuses); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.WithdrawalsBonuses)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, models.UserID) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewService(t interface {
	mock.TestingT
	Cleanup(func())
}) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}