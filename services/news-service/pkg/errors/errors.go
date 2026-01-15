package errors

import (
	"errors"
	"net/http"
)

// Error types for domain errors
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

type UnauthorizedError struct {
	Message string
}

func (e *UnauthorizedError) Error() string {
	return e.Message
}

type PermissionError struct {
	Message string
}

func (e *PermissionError) Error() string {
	return e.Message
}

type DatabaseError struct {
	Message string
}

func (e *DatabaseError) Error() string {
	return e.Message
}

// Constructors
func NewValidationError(msg string) error {
	return &ValidationError{Message: msg}
}

func NewNotFoundError(msg string) error {
	return &NotFoundError{Message: msg}
}

func NewConflictError(msg string) error {
	return &ConflictError{Message: msg}
}

func NewUnauthorizedError(msg string) error {
	return &UnauthorizedError{Message: msg}
}

func NewPermissionError(msg string) error {
	return &PermissionError{Message: msg}
}

func NewDatabaseError(msg string) error {
	return &DatabaseError{Message: msg}
}

// Type checks
func IsValidationError(err error) bool {
	var e *ValidationError
	return errors.As(err, &e)
}

func IsNotFoundError(err error) bool {
	var e *NotFoundError
	return errors.As(err, &e)
}

func IsConflictError(err error) bool {
	var e *ConflictError
	return errors.As(err, &e)
}

func IsUnauthorizedError(err error) bool {
	var e *UnauthorizedError
	return errors.As(err, &e)
}

func IsPermissionError(err error) bool {
	var e *PermissionError
	return errors.As(err, &e)
}

func IsDatabaseError(err error) bool {
	var e *DatabaseError
	return errors.As(err, &e)
}

// Mapper maps domain errors to HTTP status codes
type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

func (m *Mapper) MapErrorToHttp(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}

	switch {
	case IsValidationError(err):
		return http.StatusBadRequest, err.Error()
	case IsNotFoundError(err):
		return http.StatusNotFound, err.Error()
	case IsConflictError(err):
		return http.StatusConflict, err.Error()
	case IsUnauthorizedError(err):
		return http.StatusUnauthorized, err.Error()
	case IsPermissionError(err):
		return http.StatusForbidden, err.Error()
	case IsDatabaseError(err):
		return http.StatusInternalServerError, "internal server error"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
