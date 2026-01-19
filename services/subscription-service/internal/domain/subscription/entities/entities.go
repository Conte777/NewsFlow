package entities

import "time"

// SubscriptionStatus represents the state of a subscription in the Saga workflow
type SubscriptionStatus string

const (
	StatusPending  SubscriptionStatus = "pending"  // Waiting for activation by account-service
	StatusActive   SubscriptionStatus = "active"   // Successfully activated
	StatusRemoving SubscriptionStatus = "removing" // Waiting for removal confirmation
)

// User represents a Telegram user
type User struct {
	ID         uint      `gorm:"primaryKey"`
	TelegramID int64     `gorm:"not null;uniqueIndex"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (User) TableName() string {
	return "users"
}

// Channel represents a Telegram channel
type Channel struct {
	ID          uint      `gorm:"primaryKey"`
	ChannelID   string    `gorm:"column:channel_id;not null;uniqueIndex"`
	ChannelName string    `gorm:"column:channel_name;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (Channel) TableName() string {
	return "channels"
}

// Subscription represents a user-channel subscription (many-to-many)
type Subscription struct {
	ID        uint               `gorm:"primaryKey"`
	UserID    uint               `gorm:"not null;index"`
	ChannelID uint               `gorm:"not null;index"`
	Status    SubscriptionStatus `gorm:"type:varchar(20);default:active;index"`
	CreatedAt time.Time          `gorm:"autoCreateTime"`
	UpdatedAt time.Time          `gorm:"autoUpdateTime"`
	User      *User              `gorm:"foreignKey:UserID"`
	Channel   *Channel           `gorm:"foreignKey:ChannelID"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

// SubscriptionView is a denormalized view for API responses
type SubscriptionView struct {
	ID          uint
	TelegramID  int64
	ChannelID   string
	ChannelName string
	Status      SubscriptionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
