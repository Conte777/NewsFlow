package domain

import "errors"

var (
	// ErrNewsNotFound is returned when news is not found
	ErrNewsNotFound = errors.New("news not found")

	// ErrNewsAlreadyExists is returned when news already exists
	ErrNewsAlreadyExists = errors.New("news already exists")

	// ErrAlreadyDelivered is returned when news is already delivered to user
	ErrAlreadyDelivered = errors.New("news already delivered to user")

	// ErrInvalidChannelID is returned when channel ID is invalid
	ErrInvalidChannelID = errors.New("invalid channel ID")

	// ErrInvalidUserID is returned when user ID is invalid
	ErrInvalidUserID = errors.New("invalid user ID")

	// ErrDatabaseOperation is returned when database operation fails
	ErrDatabaseOperation = errors.New("database operation failed")
)
