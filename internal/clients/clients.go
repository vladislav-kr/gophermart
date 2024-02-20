package clients

import "errors"

var (
	ErrNotRegistered = errors.New("not registered")
	ErrInternalError = errors.New("internal error")
	ErrManyRequests  = errors.New("many requests")
)
