package errors

import "errors"

var (
	ErrSessionNotFound     = errors.New("qr auth session not found")
	ErrSessionExpired      = errors.New("qr auth session expired")
	ErrPasswordRequired    = errors.New("2fa password required")
	ErrInvalidPassword     = errors.New("invalid 2fa password")
	ErrAuthFailed          = errors.New("qr authentication failed")
	ErrAuthInProgress      = errors.New("authentication already in progress")
	ErrInvalidSessionState = errors.New("invalid session state for this operation")
	ErrMaxSessionsReached  = errors.New("maximum concurrent auth sessions reached")
	ErrQRGenerationFailed  = errors.New("failed to generate qr code")
	ErrTelegramConnection  = errors.New("failed to connect to telegram")
)
