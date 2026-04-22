package errors

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
)
