package errors

import "fmt"

type baseError struct {
	message string
}

func (e *baseError) Error() string {
	return e.message
}

// ValidationError represents a validation error (HTTP 400)
type ValidationError struct {
	baseError
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{baseError{message: message}}
}

func NewValidationErrorf(format string, args ...interface{}) *ValidationError {
	return &ValidationError{baseError{message: fmt.Sprintf(format, args...)}}
}

// UnauthorizedError represents an authentication error (HTTP 401)
type UnauthorizedError struct {
	baseError
}

func NewUnauthorizedError(message string) *UnauthorizedError {
	return &UnauthorizedError{baseError{message: message}}
}

func NewUnauthorizedErrorf(format string, args ...interface{}) *UnauthorizedError {
	return &UnauthorizedError{baseError{message: fmt.Sprintf(format, args...)}}
}

// PermissionError represents a permission error (HTTP 403)
type PermissionError struct {
	baseError
}

func NewPermissionError(message string) *PermissionError {
	return &PermissionError{baseError{message: message}}
}

func NewPermissionErrorf(format string, args ...interface{}) *PermissionError {
	return &PermissionError{baseError{message: fmt.Sprintf(format, args...)}}
}

// NotFoundError represents a not found error (HTTP 404)
type NotFoundError struct {
	baseError
}

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{baseError{message: message}}
}

func NewNotFoundErrorf(format string, args ...interface{}) *NotFoundError {
	return &NotFoundError{baseError{message: fmt.Sprintf(format, args...)}}
}

// ConflictError represents a conflict error (HTTP 409)
type ConflictError struct {
	baseError
}

func NewConflictError(message string) *ConflictError {
	return &ConflictError{baseError{message: message}}
}

func NewConflictErrorf(format string, args ...interface{}) *ConflictError {
	return &ConflictError{baseError{message: fmt.Sprintf(format, args...)}}
}

// InternalError represents an internal server error (HTTP 500)
type InternalError struct {
	baseError
}

func NewInternalError(message string) *InternalError {
	return &InternalError{baseError{message: message}}
}

func NewInternalErrorf(format string, args ...interface{}) *InternalError {
	return &InternalError{baseError{message: fmt.Sprintf(format, args...)}}
}

// ServiceUnavailableError represents a service unavailable error (HTTP 503)
type ServiceUnavailableError struct {
	baseError
}

func NewServiceUnavailableError(message string) *ServiceUnavailableError {
	return &ServiceUnavailableError{baseError{message: message}}
}

func NewServiceUnavailableErrorf(format string, args ...interface{}) *ServiceUnavailableError {
	return &ServiceUnavailableError{baseError{message: fmt.Sprintf(format, args...)}}
}
