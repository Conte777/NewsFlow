package deps

import (
	"context"

	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/entities"
)

// NewsRepository defines the interface for news data access
type NewsRepository interface {
	// Create creates a new news item
	Create(ctx context.Context, news *entities.News) error

	// GetByID retrieves news by ID
	GetByID(ctx context.Context, id uint) (*entities.News, error)

	// GetByChannelAndMessageID retrieves news by channel and message ID
	GetByChannelAndMessageID(ctx context.Context, channelID string, messageID int) (*entities.News, error)

	// Exists checks if news exists
	Exists(ctx context.Context, channelID string, messageID int) (bool, error)
}

// DeliveredNewsRepository defines the interface for delivered news data access
type DeliveredNewsRepository interface {
	// Create records that news was delivered to user
	Create(ctx context.Context, delivered *entities.DeliveredNews) error

	// IsDelivered checks if news was already delivered to user
	IsDelivered(ctx context.Context, newsID uint, userID int64) (bool, error)

	// GetUserDeliveredNews retrieves all news delivered to user
	GetUserDeliveredNews(ctx context.Context, userID int64, limit int) ([]entities.DeliveredNews, error)
}

// KafkaProducer defines interface for sending messages to Kafka
type KafkaProducer interface {
	// SendNewsDelivery sends news delivery event to bot service
	SendNewsDelivery(ctx context.Context, newsID uint, userID int64, channelID, channelName, content string, mediaURLs []string) error

	// Close closes the producer
	Close() error
}

// SubscriptionClient defines interface for interacting with subscription service
type SubscriptionClient interface {
	// GetChannelSubscribers retrieves all users subscribed to a channel
	GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error)
}
