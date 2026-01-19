package dto

import "time"

// Event types for Saga workflow
const (
	// Saga: Subscription flow
	EventTypeSubscriptionRequested = "subscription_requested"
	EventTypeSubscriptionPending   = "subscription_pending"
	EventTypeSubscriptionActivated = "subscription_activated"
	EventTypeSubscriptionFailed    = "subscription_failed"
	EventTypeSubscriptionRejected  = "subscription_rejected"

	// Saga: Subscription confirmed (to bot-service)
	EventTypeSubscriptionConfirmed = "subscription_confirmed"

	// Saga: Unsubscription flow
	EventTypeUnsubscriptionRequested  = "unsubscription_requested"
	EventTypeUnsubscriptionPending    = "unsubscription_pending"
	EventTypeUnsubscriptionCompleted  = "unsubscription_completed"
	EventTypeUnsubscriptionFailed     = "unsubscription_failed"
	EventTypeUnsubscriptionRejected   = "unsubscription_rejected"
	EventTypeUnsubscriptionConfirmed  = "unsubscription_confirmed"
)

// SubscriptionEvent is the base event structure (legacy + new)
type SubscriptionEvent struct {
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Active      bool   `json:"active,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
}

// PendingEvent is sent to account-service to trigger channel subscription/unsubscription
type PendingEvent struct {
	Type           string `json:"type"`
	SubscriptionID uint   `json:"subscription_id,omitempty"`
	UserID         string `json:"user_id"`
	ChannelID      string `json:"channel_id"`
	ChannelName    string `json:"channel_name,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`
}

// ResultEvent is received from account-service with activation/failure result
type ResultEvent struct {
	Type      string `json:"type"`
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	Reason    string `json:"reason,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// RejectedEvent is sent to bot-service to notify user about failure
type RejectedEvent struct {
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Reason      string `json:"reason"`
	Timestamp   int64  `json:"timestamp,omitempty"`
}

// ConfirmedEvent is sent to bot-service to notify user about success
type ConfirmedEvent struct {
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Timestamp   int64  `json:"timestamp,omitempty"`
}

// NewPendingEvent creates a PendingEvent with current timestamp
func NewPendingEvent(eventType string, subscriptionID uint, userID, channelID, channelName string) *PendingEvent {
	return &PendingEvent{
		Type:           eventType,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		ChannelID:      channelID,
		ChannelName:    channelName,
		Timestamp:      time.Now().Unix(),
	}
}

// NewRejectedEvent creates a RejectedEvent with current timestamp
func NewRejectedEvent(eventType string, userID, channelID, channelName, reason string) *RejectedEvent {
	return &RejectedEvent{
		Type:        eventType,
		UserID:      userID,
		ChannelID:   channelID,
		ChannelName: channelName,
		Reason:      reason,
		Timestamp:   time.Now().Unix(),
	}
}

// NewConfirmedEvent creates a ConfirmedEvent with current timestamp
func NewConfirmedEvent(eventType string, userID, channelID, channelName string) *ConfirmedEvent {
	return &ConfirmedEvent{
		Type:        eventType,
		UserID:      userID,
		ChannelID:   channelID,
		ChannelName: channelName,
		Timestamp:   time.Now().Unix(),
	}
}

type CreateSubscriptionRequest struct {
	UserID      int64  `json:"userId" validate:"required,gt=0"`
	ChannelID   string `json:"channelId" validate:"required"`
	ChannelName string `json:"channelName" validate:"required"`
}

type SubscriptionResponse struct {
	ID          uint   `json:"id"`
	UserID      int64  `json:"userId"`
	ChannelID   string `json:"channelId"`
	ChannelName string `json:"channelName"`
	IsActive    bool   `json:"isActive"`
}
