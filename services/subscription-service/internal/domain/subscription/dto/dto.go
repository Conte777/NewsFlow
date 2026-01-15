package dto

type SubscriptionEvent struct {
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Active      bool   `json:"active"`
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
