package kafka

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
