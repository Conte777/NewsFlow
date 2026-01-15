// Package deps contains interface definitions for the bot domain dependencies
package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
)

// TelegramSender defines interface for sending messages via Telegram
// This interface is used to break the cyclic dependency between UseCase and TelegramHandler
type TelegramSender interface {
	// SendMessage sends a text message to user
	SendMessage(ctx context.Context, userID int64, text string) error

	// SendMessageWithMedia sends a message with media to user
	SendMessageWithMedia(ctx context.Context, userID int64, text string, mediaURLs []string) error
}

// SubscriptionEventProducer defines interface for sending subscription events to Kafka
type SubscriptionEventProducer interface {
	// SendSubscriptionCreated sends subscription created event
	SendSubscriptionCreated(ctx context.Context, subscription *entities.Subscription) error

	// SendSubscriptionDeleted sends subscription deleted event
	SendSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error

	// Close closes the producer
	Close() error
}

// SubscriptionRepository defines interface for subscription data access
type SubscriptionRepository interface {
	// GetUserSubscriptions returns list of user subscriptions
	GetUserSubscriptions(ctx context.Context, userID int64) ([]entities.Subscription, error)

	// SaveSubscription saves a user subscription
	SaveSubscription(ctx context.Context, subscription *entities.Subscription) error

	// DeleteSubscription deletes a user subscription
	DeleteSubscription(ctx context.Context, userID int64, channelID string) error

	// CheckSubscription checks if user is subscribed to a channel
	CheckSubscription(ctx context.Context, userID int64, channelID string) (bool, error)
}
