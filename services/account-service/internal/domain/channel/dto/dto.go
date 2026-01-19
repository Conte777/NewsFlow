package dto

// SubscriptionEvent represents a subscription event from subscription service
type SubscriptionEvent struct {
	EventType   string `json:"event_type" validate:"required,oneof=created deleted"`
	UserID      int64  `json:"user_id" validate:"required"`
	ChannelID   string `json:"channel_id" validate:"required"`
	ChannelName string `json:"channel_name,omitempty"`
}

// SubscribeRequest represents a subscribe request
type SubscribeRequest struct {
	ChannelID   string `json:"channelId" validate:"required"`
	ChannelName string `json:"channelName" validate:"required"`
}

// UnsubscribeRequest represents an unsubscribe request
type UnsubscribeRequest struct {
	ChannelID string `json:"channelId" validate:"required"`
}

// ChannelResponse represents a channel in response
type ChannelResponse struct {
	ChannelID   string `json:"channelId"`
	ChannelName string `json:"channelName"`
}

// GetChannelsResponse represents get channels response
type GetChannelsResponse struct {
	Channels []ChannelResponse `json:"channels"`
	Count    int               `json:"count"`
}
