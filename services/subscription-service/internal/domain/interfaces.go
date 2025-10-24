package domain

import "context"

// SubscriptionRepository defines the interface for subscription data access
type SubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, subscription *Subscription) error

	// Delete deletes a subscription
	Delete(ctx context.Context, userID int64, channelID string) error

	// GetByUserID retrieves all subscriptions for a user
	GetByUserID(ctx context.Context, userID int64) ([]Subscription, error)

	// GetByChannelID retrieves all subscriptions for a channel
	GetByChannelID(ctx context.Context, channelID string) ([]Subscription, error)

	// Exists checks if subscription exists
	Exists(ctx context.Context, userID int64, channelID string) (bool, error)

	// GetActiveChannels retrieves all active channels
	GetActiveChannels(ctx context.Context) ([]string, error)
}

// SubscriptionUseCase defines the business logic interface for subscription operations
type SubscriptionUseCase interface {
	// CreateSubscription creates a new subscription
	CreateSubscription(ctx context.Context, userID int64, channelID, channelName string) error

	// DeleteSubscription deletes a subscription
	DeleteSubscription(ctx context.Context, userID int64, channelID string) error

	// GetUserSubscriptions retrieves all subscriptions for a user
	GetUserSubscriptions(ctx context.Context, userID int64) ([]Subscription, error)

	// GetChannelSubscribers retrieves all users subscribed to a channel
	GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error)

	// GetActiveChannels retrieves all active channels
	GetActiveChannels(ctx context.Context) ([]string, error)
}

// KafkaConsumer defines interface for receiving messages from Kafka
type KafkaConsumer interface {
	// ConsumeSubscriptionEvents consumes subscription events
	ConsumeSubscriptionEvents(ctx context.Context, handler SubscriptionEventHandler) error

	// Close closes the consumer
	Close() error
}

// KafkaProducer defines interface for sending messages to Kafka
type KafkaProducer interface {
	// NotifyAccountService notifies account service about subscription changes
	NotifyAccountService(ctx context.Context, event string, subscription *Subscription) error

	// Close closes the producer
	Close() error
}

// SubscriptionEventHandler handles subscription events
type SubscriptionEventHandler interface {
	// HandleSubscriptionCreated handles subscription created event
	HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error

	// HandleSubscriptionDeleted handles subscription deleted event
	HandleSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error
}
