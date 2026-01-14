package errors

import "errors"

var (
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrInvalidUserID             = errors.New("invalid user ID")
	ErrInvalidChannelID          = errors.New("invalid channel ID")
	ErrDatabaseOperation         = errors.New("database operation failed")
)
