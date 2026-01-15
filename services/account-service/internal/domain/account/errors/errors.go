package errors

import (
	pkgerrors "github.com/Conte777/NewsFlow/services/account-service/pkg/errors"
)

var (
	ErrAccountNotFound     = pkgerrors.NewNotFoundError("account not found")
	ErrAccountAlreadyExists = pkgerrors.NewConflictError("account already exists")
	ErrNoActiveAccounts    = pkgerrors.NewServiceUnavailableError("no active accounts available")
	ErrAuthenticationFailed = pkgerrors.NewUnauthorizedError("authentication failed")
	ErrConnectionFailed    = pkgerrors.NewInternalError("connection failed")
	ErrNotConnected        = pkgerrors.NewInternalError("not connected to Telegram")
	ErrSessionRevoked      = pkgerrors.NewUnauthorizedError("session has been revoked")
	ErrRateLimitExceeded   = pkgerrors.NewValidationError("rate limit exceeded")
	ErrPeerFlood           = pkgerrors.NewValidationError("peer flood: too many requests")
	ErrFloodWait           = pkgerrors.NewValidationError("flood wait required")
)
