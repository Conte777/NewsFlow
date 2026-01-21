// Package dto contains data transfer objects for the bot domain
package dto

import "time"

// MediaMetadata contains media type and attributes for proper rendering
type MediaMetadata struct {
	Type     string `json:"type"`               // photo, video, video_note, voice, audio, document
	Width    int    `json:"width,omitempty"`    // Video/VideoNote width
	Height   int    `json:"height,omitempty"`   // Video/VideoNote height
	Duration int    `json:"duration,omitempty"` // Video/VideoNote/Voice/Audio duration in seconds
}

// Media type constants
const (
	MediaTypePhoto     = "photo"
	MediaTypeVideo     = "video"
	MediaTypeVideoNote = "video_note"
	MediaTypeVoice     = "voice"
	MediaTypeAudio     = "audio"
	MediaTypeDocument  = "document"
)

// StartCommandRequest represents a request to handle /start command
type StartCommandRequest struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
}

// SubscribeRequest represents a request to subscribe to channels
type SubscribeRequest struct {
	UserID   int64    `json:"userId" validate:"required"`
	Channels []string `json:"channels" validate:"required,min=1"`
}

// UnsubscribeRequest represents a request to unsubscribe from channels
type UnsubscribeRequest struct {
	UserID   int64    `json:"userId" validate:"required"`
	Channels []string `json:"channels" validate:"required,min=1"`
}

// SubscriptionCreatedEvent represents a Kafka event for subscription creation
type SubscriptionCreatedEvent struct {
	UserID      int64  `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	CreatedAt   string `json:"created_at"`
}

// SubscriptionDeletedEvent represents a Kafka event for subscription deletion
type SubscriptionDeletedEvent struct {
	UserID    int64  `json:"user_id"`
	ChannelID string `json:"channel_id"`
	DeletedAt string `json:"deleted_at"`
}

// NewsDeliveryEvent represents a Kafka event for news delivery (batch format)
type NewsDeliveryEvent struct {
	NewsID        uint            `json:"news_id"`
	UserIDs       []int64         `json:"user_ids"`
	ChannelID     string          `json:"channel_id"`
	ChannelName   string          `json:"channel_name"`
	MessageID     int             `json:"message_id"` // Telegram message ID for copyMessage
	Content       string          `json:"content"`
	MediaURLs     []string        `json:"media_urls"`
	MediaMetadata []MediaMetadata `json:"media_metadata,omitempty"`
	Timestamp     int64           `json:"timestamp"`
}

// CommandResponse represents a response for bot commands
type CommandResponse struct {
	Message string `json:"message"`
}

// SubscriptionListResponse represents a response for listing subscriptions
type SubscriptionListResponse struct {
	Subscriptions []SubscriptionItem `json:"subscriptions"`
}

// SubscriptionItem represents a single subscription in the list
type SubscriptionItem struct {
	ChannelID   string    `json:"channelId"`
	ChannelName string    `json:"channelName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ToggleSubscriptionRequest represents a request to toggle subscription
type ToggleSubscriptionRequest struct {
	UserID      int64  `json:"userId" validate:"required"`
	ChannelID   string `json:"channelId" validate:"required"`
	ChannelName string `json:"channelName"`
}

// ToggleSubscriptionResponse represents response for toggle subscription
type ToggleSubscriptionResponse struct {
	Message string `json:"message"`
	Action  string `json:"action"` // "subscribed" or "unsubscribed"
}

// RejectedEvent represents a Kafka event for subscription/unsubscription rejection (Saga)
type RejectedEvent struct {
	Type        string `json:"type"` // "subscription_rejected" or "unsubscription_rejected"
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Reason      string `json:"reason"`
}

// ConfirmedEvent represents a Kafka event for subscription/unsubscription confirmation (Saga)
type ConfirmedEvent struct {
	Type        string `json:"type"` // "subscription_confirmed" or "unsubscription_confirmed"
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
}

// NewsDeleteEvent represents a Kafka event for deleting news from user chats
type NewsDeleteEvent struct {
	NewsID  uint    `json:"news_id"`
	UserIDs []int64 `json:"user_ids"`
}

// NewsEditEvent represents a Kafka event for editing news in user chats
type NewsEditEvent struct {
	NewsID        uint            `json:"news_id"`
	UserIDs       []int64         `json:"user_ids"`
	Content       string          `json:"content"`
	ChannelName   string          `json:"channel_name"`
	MediaURLs     []string        `json:"media_urls"`
	MediaMetadata []MediaMetadata `json:"media_metadata,omitempty"`
}
