// Code generated by mockery v2.37.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// PasswordGenerator is an autogenerated mock type for the PasswordGenerator type
type PasswordGenerator struct {
	mock.Mock
}

// CompareHashAndPassword provides a mock function with given fields: hashedPassword, password
func (_m *PasswordGenerator) CompareHashAndPassword(hashedPassword []byte, password []byte) error {
	ret := _m.Called(hashedPassword, password)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, []byte) error); ok {
		r0 = rf(hashedPassword, password)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GenerateFromPassword provides a mock function with given fields: password
func (_m *PasswordGenerator) GenerateFromPassword(password []byte) ([]byte, error) {
	ret := _m.Called(password)

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) ([]byte, error)); ok {
		return rf(password)
	}
	if rf, ok := ret.Get(0).(func([]byte) []byte); ok {
		r0 = rf(password)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewPasswordGenerator creates a new instance of PasswordGenerator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPasswordGenerator(t interface {
	mock.TestingT
	Cleanup(func())
}) *PasswordGenerator {
	mock := &PasswordGenerator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}