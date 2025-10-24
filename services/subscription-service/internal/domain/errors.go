package domain

import "errors"

var (
	// ErrSubscriptionNotFound is returned when subscription is not found
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrSubscriptionAlreadyExists is returned when subscription already exists
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")

	// ErrInvalidUserID is returned when user ID is invalid
	ErrInvalidUserID = errors.New("invalid user ID")

	// ErrInvalidChannelID is returned when channel ID is invalid
	ErrInvalidChannelID = errors.New("invalid channel ID")

	// ErrDatabaseOperation is returned when database operation fails
	ErrDatabaseOperation = errors.New("database operation failed")
)
