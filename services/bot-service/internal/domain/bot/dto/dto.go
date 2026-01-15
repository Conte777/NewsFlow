// Package dto contains data transfer objects for the bot domain
package dto

import "time"

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

// NewsDeliveryEvent represents a Kafka event for news delivery
type NewsDeliveryEvent struct {
	ID          string   `json:"id"`
	UserID      int64    `json:"user_id"`
	ChannelID   string   `json:"channel_id"`
	ChannelName string   `json:"channel_name"`
	Content     string   `json:"content"`
	MediaURLs   []string `json:"media_urls"`
	CreatedAt   string   `json:"created_at"`
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
