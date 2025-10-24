package domain

import "context"

// NewsRepository defines the interface for news data access
type NewsRepository interface {
	// Create creates a new news item
	Create(ctx context.Context, news *News) error

	// GetByID retrieves news by ID
	GetByID(ctx context.Context, id uint) (*News, error)

	// GetByChannelAndMessageID retrieves news by channel and message ID
	GetByChannelAndMessageID(ctx context.Context, channelID string, messageID int) (*News, error)

	// Exists checks if news exists
	Exists(ctx context.Context, channelID string, messageID int) (bool, error)
}

// DeliveredNewsRepository defines the interface for delivered news data access
type DeliveredNewsRepository interface {
	// Create records that news was delivered to user
	Create(ctx context.Context, delivered *DeliveredNews) error

	// IsDelivered checks if news was already delivered to user
	IsDelivered(ctx context.Context, newsID uint, userID int64) (bool, error)

	// GetUserDeliveredNews retrieves all news delivered to user
	GetUserDeliveredNews(ctx context.Context, userID int64, limit int) ([]DeliveredNews, error)
}

// NewsUseCase defines the business logic interface for news operations
type NewsUseCase interface {
	// ProcessNewsReceived processes a news received event
	ProcessNewsReceived(ctx context.Context, channelID, channelName string, messageID int, content string, mediaURLs []string) error

	// DeliverNewsToUsers delivers news to subscribed users
	DeliverNewsToUsers(ctx context.Context, newsID uint, userIDs []int64) error

	// MarkAsDelivered marks news as delivered to user
	MarkAsDelivered(ctx context.Context, newsID uint, userID int64) error

	// GetUserDeliveredNews retrieves user's delivered news history
	GetUserDeliveredNews(ctx context.Context, userID int64, limit int) ([]DeliveredNews, error)
}

// KafkaConsumer defines interface for receiving messages from Kafka
type KafkaConsumer interface {
	// ConsumeNewsReceived consumes news received events from account service
	ConsumeNewsReceived(ctx context.Context, handler NewsReceivedHandler) error

	// Close closes the consumer
	Close() error
}

// KafkaProducer defines interface for sending messages to Kafka
type KafkaProducer interface {
	// SendNewsDelivery sends news delivery event to bot service
	SendNewsDelivery(ctx context.Context, newsID uint, userID int64, channelID, channelName, content string, mediaURLs []string) error

	// Close closes the producer
	Close() error
}

// NewsReceivedHandler handles news received events
type NewsReceivedHandler interface {
	// HandleNewsReceived handles a news received event
	HandleNewsReceived(ctx context.Context, channelID, channelName string, messageID int, content string, mediaURLs []string) error
}

// SubscriptionService defines interface for interacting with subscription service
type SubscriptionService interface {
	// GetChannelSubscribers retrieves all users subscribed to a channel
	GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error)
}
