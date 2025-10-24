package domain

import (
	"time"
)

// Subscription represents a user subscription to a channel
type Subscription struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      int64     `gorm:"not null;index:idx_user_channel,unique"`
	ChannelID   string    `gorm:"not null;index:idx_user_channel,unique;index:idx_channel"`
	ChannelName string    `gorm:"not null"`
	IsActive    bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}
