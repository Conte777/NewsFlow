package domain

import "errors"

var (
	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidChannelID is returned when channel ID is invalid
	ErrInvalidChannelID = errors.New("invalid channel ID")

	// ErrSubscriptionNotFound is returned when subscription is not found
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrAlreadySubscribed is returned when user is already subscribed
	ErrAlreadySubscribed = errors.New("already subscribed")

	// ErrMessageDeliveryFailed is returned when message delivery fails
	ErrMessageDeliveryFailed = errors.New("message delivery failed")

	// ErrInvalidCommand is returned when command is invalid
	ErrInvalidCommand = errors.New("invalid command")

	ErrTelegramAPI = errors.New("telegram API error")
)
