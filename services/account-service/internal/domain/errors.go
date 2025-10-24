package domain

import "errors"

var (
	// ErrAccountNotFound is returned when account is not found
	ErrAccountNotFound = errors.New("account not found")

	// ErrAccountAlreadyExists is returned when account already exists
	ErrAccountAlreadyExists = errors.New("account already exists")

	// ErrChannelNotFound is returned when channel is not found
	ErrChannelNotFound = errors.New("channel not found")

	// ErrInvalidChannelID is returned when channel ID is invalid
	ErrInvalidChannelID = errors.New("invalid channel ID")

	// ErrSubscriptionFailed is returned when subscription fails
	ErrSubscriptionFailed = errors.New("subscription failed")

	// ErrUnsubscriptionFailed is returned when unsubscription fails
	ErrUnsubscriptionFailed = errors.New("unsubscription failed")

	// ErrNoActiveAccounts is returned when no active accounts available
	ErrNoActiveAccounts = errors.New("no active accounts available")

	// ErrNewsCollectionFailed is returned when news collection fails
	ErrNewsCollectionFailed = errors.New("news collection failed")
)
