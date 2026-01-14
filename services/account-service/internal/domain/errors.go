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

	// ErrAuthenticationFailed is returned when authentication fails
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrConnectionFailed is returned when connection to Telegram fails
	ErrConnectionFailed = errors.New("connection failed")

	// ErrNotConnected is returned when operation requires connection
	ErrNotConnected = errors.New("not connected to Telegram")

	// ErrSessionRevoked is returned when session is revoked
	ErrSessionRevoked = errors.New("session has been revoked")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrChannelPrivate is returned when trying to access a private channel
	ErrChannelPrivate = errors.New("channel is private")

	// ErrPeerFlood is returned when too many requests are made (anti-spam)
	ErrPeerFlood = errors.New("peer flood: too many requests, please wait")

	// ErrFloodWait is returned when flood wait is required
	ErrFloodWait = errors.New("flood wait required")
)
