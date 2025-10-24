package domain

import "time"

// TelegramAccount represents a Telegram account managed by the service
type TelegramAccount struct {
	ID          string
	PhoneNumber string
	IsActive    bool
	CreatedAt   time.Time
}

// ChannelSubscription represents a channel subscription for an account
type ChannelSubscription struct {
	AccountID   string
	ChannelID   string
	ChannelName string
	IsActive    bool
	CreatedAt   time.Time
}

// NewsItem represents a news message from a channel
type NewsItem struct {
	ChannelID   string
	ChannelName string
	MessageID   int
	Content     string
	MediaURLs   []string
	Date        time.Time
}

// SubscriptionEvent represents a subscription event from subscription service
type SubscriptionEvent struct {
	EventType   string // "subscription.created" or "subscription.deleted"
	UserID      int64
	ChannelID   string
	ChannelName string
}
