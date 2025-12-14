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
// JSON tags are added to match the event format specification (ACC-2.2)
type NewsItem struct {
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	MessageID   int       `json:"message_id"`
	Content     string    `json:"content"`
	MediaURLs   []string  `json:"media_urls"`
	Date        time.Time `json:"date"`
}

// SubscriptionEvent represents a subscription event from subscription service
type SubscriptionEvent struct {
	EventType   string `json:"event_type"`
	UserID      int64  `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name,omitempty"` // omitempty for deleted events
}

// ChannelInfo represents detailed information about a Telegram channel
type ChannelInfo struct {
	ID               string
	Username         string
	Title            string
	About            string // Channel description
	ParticipantsCount int
	PhotoURL         string // Channel photo URL (if available)
	IsVerified       bool
	IsRestricted     bool
	CreatedAt        time.Time
}
