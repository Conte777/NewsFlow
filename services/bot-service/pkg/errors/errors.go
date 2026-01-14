// Package errors provides typed errors for the application
package errors

// ErrorType represents the type of error
type ErrorType int

const (
	ErrorTypeValidation ErrorType = iota
	ErrorTypeNotFound
	ErrorTypeConflict
	ErrorTypeUnauthorized
	ErrorTypePermission
	ErrorTypeInternal
)

// baseError is the base implementation for all error types
type baseError struct {
	msg string
}

func (e *baseError) Error() string {
	return e.msg
}

// ValidationError represents a validation error (400)
type ValidationError struct {
	baseError
}

// NewValidationError creates a new ValidationError
func NewValidationError(msg string) *ValidationError {
	return &ValidationError{baseError{msg: msg}}
}

// NotFoundError represents a not found error (404)
type NotFoundError struct {
	baseError
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(msg string) *NotFoundError {
	return &NotFoundError{baseError{msg: msg}}
}

// ConflictError represents a conflict error (409)
type ConflictError struct {
	baseError
}

// NewConflictError creates a new ConflictError
func NewConflictError(msg string) *ConflictError {
	return &ConflictError{baseError{msg: msg}}
}

// UnauthorizedError represents an unauthorized error (401)
type UnauthorizedError struct {
	baseError
}

// NewUnauthorizedError creates a new UnauthorizedError
func NewUnauthorizedError(msg string) *UnauthorizedError {
	return &UnauthorizedError{baseError{msg: msg}}
}

// PermissionError represents a permission error (403)
type PermissionError struct {
	baseError
}

// NewPermissionError creates a new PermissionError
func NewPermissionError(msg string) *PermissionError {
	return &PermissionError{baseError{msg: msg}}
}

// InternalError represents an internal server error (500)
type InternalError struct {
	baseError
}

// NewInternalError creates a new InternalError
func NewInternalError(msg string) *InternalError {
	return &InternalError{baseError{msg: msg}}
}

// IsValidationError checks if error is a ValidationError
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsNotFoundError checks if error is a NotFoundError
func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// IsConflictError checks if error is a ConflictError
func IsConflictError(err error) bool {
	_, ok := err.(*ConflictError)
	return ok
}

// IsUnauthorizedError checks if error is an UnauthorizedError
func IsUnauthorizedError(err error) bool {
	_, ok := err.(*UnauthorizedError)
	return ok
}

// IsPermissionError checks if error is a PermissionError
func IsPermissionError(err error) bool {
	_, ok := err.(*PermissionError)
	return ok
}

// IsInternalError checks if error is an InternalError
func IsInternalError(err error) bool {
	_, ok := err.(*InternalError)
	return ok
}
