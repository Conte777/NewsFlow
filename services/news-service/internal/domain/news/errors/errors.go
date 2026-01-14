package errors

import (
	pkgerrors "github.com/yourusername/telegram-news-feed/news-service/pkg/errors"
)

var (
	// ErrNewsNotFound is returned when news is not found
	ErrNewsNotFound = pkgerrors.NewNotFoundError("news not found")

	// ErrNewsAlreadyExists is returned when news already exists
	ErrNewsAlreadyExists = pkgerrors.NewConflictError("news already exists")

	// ErrAlreadyDelivered is returned when news is already delivered to user
	ErrAlreadyDelivered = pkgerrors.NewConflictError("news already delivered to user")

	// ErrInvalidChannelID is returned when channel ID is invalid
	ErrInvalidChannelID = pkgerrors.NewValidationError("invalid channel ID")

	// ErrInvalidUserID is returned when user ID is invalid
	ErrInvalidUserID = pkgerrors.NewValidationError("invalid user ID")

	// ErrDatabaseOperation is returned when database operation fails
	ErrDatabaseOperation = pkgerrors.NewDatabaseError("database operation failed")
)
