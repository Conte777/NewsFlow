package entities

import "time"

// ChannelSubscription represents a channel subscription for an account
type ChannelSubscription struct {
	AccountID              string    `db:"account_id" json:"accountId"`
	ChannelID              string    `db:"channel_id" json:"channelId"`
	ChannelName            string    `db:"channel_name" json:"channelName"`
	IsActive               bool      `db:"is_active" json:"isActive"`
	LastProcessedMessageID int       `db:"last_processed_message_id" json:"lastProcessedMessageId"`
	CreatedAt              time.Time `db:"created_at" json:"createdAt"`
}

// ChannelInfo represents detailed information about a Telegram channel
type ChannelInfo struct {
	ID                string    `json:"id"`
	Username          string    `json:"username"`
	Title             string    `json:"title"`
	About             string    `json:"about"`
	ParticipantsCount int       `json:"participantsCount"`
	PhotoURL          string    `json:"photoUrl"`
	IsVerified        bool      `json:"isVerified"`
	IsRestricted      bool      `json:"isRestricted"`
	CreatedAt         time.Time `json:"createdAt"`
}
