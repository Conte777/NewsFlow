package kafka

import (
	"strconv"
	"time"
)

// Legacy events (for backward compatibility)

// SubscriptionCreatedEvent represents a subscription created event from bot-service
type SubscriptionCreatedEvent struct {
	UserID      int64  `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	CreatedAt   string `json:"created_at"`
}

// SubscriptionDeletedEvent represents a subscription deleted event from bot-service
type SubscriptionDeletedEvent struct {
	UserID    int64  `json:"user_id"`
	ChannelID string `json:"channel_id"`
	DeletedAt string `json:"deleted_at"`
}

// Saga events

// Event types for Saga workflow
const (
	EventTypeSubscriptionPending   = "subscription_pending"
	EventTypeSubscriptionActivated = "subscription_activated"
	EventTypeSubscriptionFailed    = "subscription_failed"

	EventTypeUnsubscriptionPending   = "unsubscription_pending"
	EventTypeUnsubscriptionCompleted = "unsubscription_completed"
	EventTypeUnsubscriptionFailed    = "unsubscription_failed"
)

// SubscriptionPendingEvent is received from subscription-service
type SubscriptionPendingEvent struct {
	Type           string `json:"type"`
	SubscriptionID uint   `json:"subscription_id,omitempty"`
	UserID         string `json:"user_id"`
	ChannelID      string `json:"channel_id"`
	ChannelName    string `json:"channel_name,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`
}

// GetUserIDInt64 parses UserID string to int64
func (e *SubscriptionPendingEvent) GetUserIDInt64() (int64, error) {
	return strconv.ParseInt(e.UserID, 10, 64)
}

// UnsubscriptionPendingEvent is received from subscription-service
type UnsubscriptionPendingEvent struct {
	Type      string `json:"type"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// GetUserIDInt64 parses UserID string to int64
func (e *UnsubscriptionPendingEvent) GetUserIDInt64() (int64, error) {
	return strconv.ParseInt(e.UserID, 10, 64)
}

// ResultEvent is sent to subscription-service with activation/failure result
type ResultEvent struct {
	Type      string `json:"type"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	Reason    string `json:"reason,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// NewSubscriptionActivatedEvent creates a subscription activated event
func NewSubscriptionActivatedEvent(userID int64, channelID string) *ResultEvent {
	return &ResultEvent{
		Type:      EventTypeSubscriptionActivated,
		UserID:    strconv.FormatInt(userID, 10),
		ChannelID: channelID,
		Timestamp: time.Now().Unix(),
	}
}

// NewSubscriptionFailedEvent creates a subscription failed event
func NewSubscriptionFailedEvent(userID int64, channelID, reason string) *ResultEvent {
	return &ResultEvent{
		Type:      EventTypeSubscriptionFailed,
		UserID:    strconv.FormatInt(userID, 10),
		ChannelID: channelID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	}
}

// NewUnsubscriptionCompletedEvent creates an unsubscription completed event
func NewUnsubscriptionCompletedEvent(userID int64, channelID string) *ResultEvent {
	return &ResultEvent{
		Type:      EventTypeUnsubscriptionCompleted,
		UserID:    strconv.FormatInt(userID, 10),
		ChannelID: channelID,
		Timestamp: time.Now().Unix(),
	}
}

// NewUnsubscriptionFailedEvent creates an unsubscription failed event
func NewUnsubscriptionFailedEvent(userID int64, channelID, reason string) *ResultEvent {
	return &ResultEvent{
		Type:      EventTypeUnsubscriptionFailed,
		UserID:    strconv.FormatInt(userID, 10),
		ChannelID: channelID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	}
}

// News sync events (delete/edit)

// NewsDeletedEvent is sent when messages are deleted from a channel
type NewsDeletedEvent struct {
	ChannelID  string `json:"channel_id"`
	MessageIDs []int  `json:"message_ids"`
	DeletedAt  int64  `json:"deleted_at"`
}

// NewsEditedEvent is sent when a message is edited in a channel
type NewsEditedEvent struct {
	ChannelID   string   `json:"channel_id"`
	ChannelName string   `json:"channel_name"`
	MessageID   int      `json:"message_id"`
	Content     string   `json:"content"`
	MediaURLs   []string `json:"media_urls"`
	EditedAt    int64    `json:"edited_at"`
}
