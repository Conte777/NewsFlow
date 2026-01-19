// Package entities contains domain entities
package entities

import "time"

// User represents a Telegram user
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	FirstName string    `json:"firstName" db:"first_name"`
	LastName  string    `json:"lastName" db:"last_name"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// Subscription represents a user subscription to a channel
type Subscription struct {
	UserID      int64     `json:"userId" db:"user_id"`
	ChannelID   string    `json:"channelId" db:"channel_id"`
	ChannelName string    `json:"channelName" db:"channel_name"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// NewsMessage represents a news message to be delivered
type NewsMessage struct {
	ID          uint      `json:"id" db:"id"`
	UserID      int64     `json:"userId" db:"user_id"`
	ChannelID   string    `json:"channelId" db:"channel_id"`
	ChannelName string    `json:"channelName" db:"channel_name"`
	Content     string    `json:"content" db:"content"`
	MediaURLs   []string  `json:"mediaUrls" db:"-"`
	MessageID   int       `json:"messageId" db:"message_id"`
	Timestamp   int64     `json:"timestamp" db:"timestamp"`
}

// DeliveredMessage tracks delivered messages to users for delete/edit sync
type DeliveredMessage struct {
	ID                uint      `gorm:"primaryKey" db:"id" json:"id"`
	NewsID            uint      `gorm:"not null;uniqueIndex:idx_news_user" db:"news_id" json:"newsId"`
	UserID            int64     `gorm:"not null;uniqueIndex:idx_news_user;index" db:"user_id" json:"userId"`
	TelegramMessageID int       `gorm:"not null" db:"telegram_message_id" json:"telegramMessageId"`
	CreatedAt         time.Time `gorm:"autoCreateTime" db:"created_at" json:"createdAt"`
}

// TableName returns the table name for DeliveredMessage
func (DeliveredMessage) TableName() string {
	return "delivered_messages"
}
