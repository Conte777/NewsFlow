package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/entities"
)

// MessageIDCache provides in-memory cache for LastProcessedMessageID
// to prevent race conditions between real-time handler and fallback collector
type MessageIDCache interface {
	// Get returns the cached lastProcessedMessageID for a channel
	// Returns (messageID, true) if found, (0, false) if not cached
	Get(channelID string) (int, bool)

	// SetIfGreater atomically updates the cache only if newMessageID > current
	// Returns true if the value was updated
	SetIfGreater(channelID string, newMessageID int) bool

	// Delete removes a channel from the cache
	Delete(channelID string)

	// LoadFromDB loads all channel message IDs from the database into the cache
	LoadFromDB(ctx context.Context) error
}

// ChannelRepository defines interface for channel subscription storage
type ChannelRepository interface {
	RemoveChannel(ctx context.Context, channelID string) error
	GetAllChannels(ctx context.Context) ([]entities.ChannelSubscription, error)
	GetChannel(ctx context.Context, channelID string) (*entities.ChannelSubscription, error)
	ChannelExists(ctx context.Context, channelID string) (bool, error)
	UpdateLastProcessedMessageID(ctx context.Context, channelID string, messageID int) error

	// New methods for account-channel binding
	AddChannelForAccount(ctx context.Context, phoneNumber, channelID, channelName string) error
	RemoveChannelForAccount(ctx context.Context, phoneNumber, channelID string) error
	GetChannelsByAccount(ctx context.Context, phoneNumber string) ([]entities.ChannelSubscription, error)
	GetAccountPhoneForChannel(ctx context.Context, channelID string) (string, error)
}

// SagaEventHandler handles Saga workflow events
type SagaEventHandler interface {
	// Subscription flow
	HandleSubscriptionPending(ctx context.Context, userID int64, channelID, channelName string) error
	// Unsubscription flow
	HandleUnsubscriptionPending(ctx context.Context, userID int64, channelID string) error
}

// SagaProducer sends Saga result events to subscription-service
type SagaProducer interface {
	// Subscription flow results
	SendSubscriptionActivated(ctx context.Context, userID int64, channelID string) error
	SendSubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error
	// Unsubscription flow results
	SendUnsubscriptionCompleted(ctx context.Context, userID int64, channelID string) error
	SendUnsubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error
	Close() error
}
