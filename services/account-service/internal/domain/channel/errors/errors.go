package errors

import (
	pkgerrors "github.com/Conte777/NewsFlow/services/account-service/pkg/errors"
)

var (
	ErrChannelNotFound       = pkgerrors.NewNotFoundError("channel not found")
	ErrInvalidChannelID      = pkgerrors.NewValidationError("invalid channel ID")
	ErrSubscriptionFailed    = pkgerrors.NewInternalError("subscription failed")
	ErrUnsubscriptionFailed  = pkgerrors.NewInternalError("unsubscription failed")
	ErrChannelPrivate        = pkgerrors.NewPermissionError("channel is private")
	ErrChannelAlreadyExists  = pkgerrors.NewConflictError("channel already subscribed")
)
