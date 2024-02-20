package models

import "errors"

var (
	ErrLoginAlreadyExists   = errors.New("login already exists")
	ErrIncorrectCredentials = errors.New("incorrect credentials")
	ErrIncorrectOrderNumber = errors.New("incorrect order number")

	ErrInternal                   = errors.New("internal Error")
	ErrAlreadyUploadedUser        = errors.New("already uploaded by user")
	ErrAlreadyUploadedAnotherUser = errors.New("already uploaded by another user")
	ErrNoRecordsFound             = errors.New("no records found")
	ErrInsufficientFunds          = errors.New("insufficient funds")

	ErrUserIDMandatory           = errors.New("userID is a mandatory parameter")
	ErrMismatchedHashAndPassword = errors.New("hashedPassword is not the hash of the given password")
)
