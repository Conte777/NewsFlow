package errors

import (
	"errors"

	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
)

// Mapper maps domain errors to HTTP status codes
type Mapper struct {
	logger zerolog.Logger
}

// NewMapper creates a new error mapper
func NewMapper(logger zerolog.Logger) *Mapper {
	return &Mapper{logger: logger}
}

// MapErrorToHTTP maps an error to HTTP status code and message
func (m *Mapper) MapErrorToHTTP(err error) (int, string) {
	if err == nil {
		return fasthttp.StatusOK, ""
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return fasthttp.StatusBadRequest, validationErr.Error()
	}

	var unauthorizedErr *UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return fasthttp.StatusUnauthorized, unauthorizedErr.Error()
	}

	var permissionErr *PermissionError
	if errors.As(err, &permissionErr) {
		return fasthttp.StatusForbidden, permissionErr.Error()
	}

	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return fasthttp.StatusNotFound, notFoundErr.Error()
	}

	var conflictErr *ConflictError
	if errors.As(err, &conflictErr) {
		return fasthttp.StatusConflict, conflictErr.Error()
	}

	var serviceUnavailableErr *ServiceUnavailableError
	if errors.As(err, &serviceUnavailableErr) {
		return fasthttp.StatusServiceUnavailable, serviceUnavailableErr.Error()
	}

	var internalErr *InternalError
	if errors.As(err, &internalErr) {
		m.logger.Error().Err(err).Msg("internal server error")
		return fasthttp.StatusInternalServerError, internalErr.Error()
	}

	m.logger.Error().Err(err).Msg("unknown error")
	return fasthttp.StatusInternalServerError, "internal server error"
}
