package domain

import "context"

// AccountUseCase defines the business logic interface for account operations
type AccountUseCase interface {
	// SubscribeToChannel subscribes an account to a channel
	SubscribeToChannel(ctx context.Context, channelID, channelName string) error

	// UnsubscribeFromChannel unsubscribes an account from a channel
	UnsubscribeFromChannel(ctx context.Context, channelID string) error

	// CollectNews collects news from all subscribed channels
	CollectNews(ctx context.Context) error

	// GetActiveChannels returns all channels that are subscribed
	GetActiveChannels(ctx context.Context) ([]string, error)
}

// TelegramClient defines interface for Telegram MTProto operations
type TelegramClient interface {
	// Connect connects to Telegram
	Connect(ctx context.Context) error

	// Disconnect disconnects from Telegram
	// The context controls the timeout for graceful shutdown
	Disconnect(ctx context.Context) error

	// JoinChannel joins a Telegram channel
	JoinChannel(ctx context.Context, channelID string) error

	// LeaveChannel leaves a Telegram channel
	LeaveChannel(ctx context.Context, channelID string) error

	// GetChannelMessages retrieves recent messages from a channel with pagination support
	GetChannelMessages(ctx context.Context, channelID string, limit, offset int) ([]NewsItem, error)

	// GetChannelInfo retrieves detailed information about a channel
	GetChannelInfo(ctx context.Context, channelID string) (*ChannelInfo, error)

	// IsConnected checks if client is connected
	IsConnected() bool

	// GetAccountID returns unique identifier for this account (e.g., phone number)
	GetAccountID() string
}

// AccountManager manages multiple Telegram accounts
type AccountManager interface {
	// GetAvailableAccount returns an available account for operation
	GetAvailableAccount() (TelegramClient, error)

	// GetAllAccounts returns all managed accounts
	GetAllAccounts() []TelegramClient

	// AddAccount adds a new account
	AddAccount(client TelegramClient) error

	// RemoveAccount removes an account
	RemoveAccount(accountID string) error
}

// ChannelRepository defines interface for channel subscription storage
type ChannelRepository interface {
	// AddChannel adds a channel subscription
	AddChannel(ctx context.Context, channelID, channelName string) error

	// RemoveChannel removes a channel subscription
	RemoveChannel(ctx context.Context, channelID string) error

	// GetAllChannels retrieves all subscribed channels
	GetAllChannels(ctx context.Context) ([]ChannelSubscription, error)

	// GetChannel retrieves a specific channel
	GetChannel(ctx context.Context, channelID string) (*ChannelSubscription, error)

	// ChannelExists checks if channel exists
	ChannelExists(ctx context.Context, channelID string) (bool, error)
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
	// SendNewsReceived sends news received event to news service
	SendNewsReceived(ctx context.Context, news *NewsItem) error

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
