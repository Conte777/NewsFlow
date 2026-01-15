// Package errors contains domain-specific errors for the bot domain
package errors

import (
	pkgerrors "github.com/Conte777/NewsFlow/services/bot-service/pkg/errors"
)

// Domain errors for bot operations
var (
	ErrUserNotFound          = pkgerrors.NewNotFoundError("user not found")
	ErrInvalidChannelID      = pkgerrors.NewValidationError("invalid channel ID format, must start with @")
	ErrSubscriptionNotFound  = pkgerrors.NewNotFoundError("subscription not found")
	ErrAlreadySubscribed     = pkgerrors.NewConflictError("already subscribed to this channel")
	ErrMessageDeliveryFailed = pkgerrors.NewInternalError("message delivery failed")
	ErrInvalidCommand        = pkgerrors.NewValidationError("invalid command")
	ErrTelegramAPI           = pkgerrors.NewInternalError("telegram API error")
	ErrEmptyMessage          = pkgerrors.NewValidationError("message text cannot be empty")
	ErrNoChannelsSpecified   = pkgerrors.NewValidationError("no channels specified")
	ErrKafkaProducer         = pkgerrors.NewInternalError("kafka producer error")
	ErrKafkaConsumer         = pkgerrors.NewInternalError("kafka consumer error")
)
