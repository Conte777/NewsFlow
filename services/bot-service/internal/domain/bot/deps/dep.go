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

	// SendChatAction sends typing indicator or other chat action
	SendChatAction(ctx context.Context, userID int64, action string) error

	// SendMessageAndGetID sends a text message to user and returns the telegram message ID
	SendMessageAndGetID(ctx context.Context, userID int64, text string) (messageID int, err error)

	// SendMessageWithMediaAndGetID sends a message with media to user and returns the telegram message ID
	SendMessageWithMediaAndGetID(ctx context.Context, userID int64, text string, mediaURLs []string) (messageID int, err error)

	// SendMessageWithFilesAndGetID sends downloaded files to user and returns the telegram message ID
	SendMessageWithFilesAndGetID(ctx context.Context, userID int64, text string, files []*entities.DownloadedFile) (messageID int, err error)

	// DownloadFiles downloads files from S3 URLs
	DownloadFiles(ctx context.Context, urls []string) ([]*entities.DownloadedFile, error)

	// CopyMessageAndGetID copies a message from a channel to user's chat and returns the telegram message ID
	// Uses Bot API copyMessage method - works for public channels
	CopyMessageAndGetID(ctx context.Context, userID int64, fromChannelID string, messageID int, caption string) (copiedMessageID int, err error)

	// DeleteMessage deletes a message from user's chat
	DeleteMessage(ctx context.Context, userID int64, messageID int) error

	// EditMessageText edits message text in user's chat
	EditMessageText(ctx context.Context, userID int64, messageID int, text string) error
}

// SubscriptionEventProducer defines interface for sending subscription events to Kafka
type SubscriptionEventProducer interface {
	// SendSubscriptionCreated sends subscription created event
	SendSubscriptionCreated(ctx context.Context, subscription *entities.Subscription) error

	// SendSubscriptionDeleted sends subscription deleted event
	SendSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error

	// SendDeliveryConfirmation sends delivery confirmation event to news-service
	SendDeliveryConfirmation(ctx context.Context, newsID uint, userID int64) error

	// Close closes the producer
	Close() error
}

// SubscriptionRepository defines interface for subscription data access (gRPC client)
type SubscriptionRepository interface {
	// GetUserSubscriptions returns list of user subscriptions via gRPC
	GetUserSubscriptions(ctx context.Context, userID int64) ([]entities.Subscription, error)
}

// DeliveredMessageRepository defines interface for delivered message data access
type DeliveredMessageRepository interface {
	// Save saves a delivered message record
	Save(ctx context.Context, msg *entities.DeliveredMessage) error

	// GetByNewsIDAndUserID retrieves a delivered message by news ID and user ID
	GetByNewsIDAndUserID(ctx context.Context, newsID uint, userID int64) (*entities.DeliveredMessage, error)

	// GetByNewsID retrieves all delivered messages for a news item
	GetByNewsID(ctx context.Context, newsID uint) ([]entities.DeliveredMessage, error)

	// Delete deletes a delivered message record
	Delete(ctx context.Context, newsID uint, userID int64) error
}
