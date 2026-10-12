package domain

import "errors"

var (
	ErrNotFound               = errors.New("not found")
	ErrConflict               = errors.New("a record with this value already exists")
	ErrReferenceNotFound      = errors.New("referenced record does not exist")
	ErrInvalidCredentials     = errors.New("invalid username or password")
	ErrUnauthenticated        = errors.New("not authenticated")
	ErrPasswordChangeRequired = errors.New("PASSWORD_CHANGE_REQUIRED")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + " " + e.Message }

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
