package domain

import "context"

// BotUseCase defines the business logic interface for bot operations
type BotUseCase interface {
	// HandleStart handles /start command
	HandleStart(ctx context.Context, userID int64, username string) (string, error)

	// HandleHelp handles /help command
	HandleHelp(ctx context.Context) (string, error)

	// HandleSubscribe handles subscription request
	HandleSubscribe(ctx context.Context, userID int64, channelID string) (string, error)

	// HandleUnsubscribe handles unsubscription request
	HandleUnsubscribe(ctx context.Context, userID int64, channelID string) (string, error)

	// HandleListSubscriptions handles listing user subscriptions
	HandleListSubscriptions(ctx context.Context, userID int64) ([]Subscription, error)

	// SendNews sends news to user
	SendNews(ctx context.Context, news *NewsMessage) error

	// Новый метод для установки TelegramBot
	SetTelegramBot(bot TelegramBot)
}

// KafkaProducer defines interface for sending messages to Kafka
type KafkaProducer interface {
	// SendSubscriptionCreated sends subscription created event
	SendSubscriptionCreated(ctx context.Context, subscription *Subscription) error

	// SendSubscriptionDeleted sends subscription deleted event
	SendSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error

	// Close closes the producer
	Close() error
}

// KafkaConsumer defines interface for receiving messages from Kafka
type KafkaConsumer interface {
	// ConsumeNewsDelivery consumes news delivery messages
	ConsumeNewsDelivery(ctx context.Context, handler func(*NewsMessage) error) error

	// Close closes the consumer
	Close() error
}

// SubscriptionRepository определяет интерфейс для работы с подписками
type SubscriptionRepository interface {
	// GetUserSubscriptions возвращает список подписок пользователя
	GetUserSubscriptions(ctx context.Context, userID int64) ([]Subscription, error)

	// SaveSubscription сохраняет подписку пользователя
	SaveSubscription(ctx context.Context, subscription *Subscription) error

	// DeleteSubscription удаляет подписку пользователя
	DeleteSubscription(ctx context.Context, userID int64, channelID string) error

	// CheckSubscription проверяет, подписан ли пользователь на канал
	CheckSubscription(ctx context.Context, userID int64, channelID string) (bool, error)
}

// SubscriptionEvent представляет событие подписки/отписки
type SubscriptionEvent struct {
	UserID    int64    `json:"user_id"`
	Channels  []string `json:"channels"`
	EventType string   `json:"event_type"` // "subscribe" или "unsubscribe"
	Timestamp int64    `json:"timestamp"`
}

// TelegramBot defines interface for Telegram bot operations
type TelegramBot interface {
	// SendMessage sends a text message to user
	SendMessage(ctx context.Context, userID int64, text string) error

	// SendMessageWithMedia sends a message with media to user
	SendMessageWithMedia(ctx context.Context, userID int64, text string, mediaURLs []string) error

	// Start starts the bot
	Start(ctx context.Context) error

	// Stop stops the bot
	Stop() error
}
