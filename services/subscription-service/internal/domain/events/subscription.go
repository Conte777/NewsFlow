package events

type SubscriptionEvent struct {
	Type        string `json:"type"`
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Active      bool   `json:"active"`
}
