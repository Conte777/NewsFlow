package domain

import (
	"time"

	"gorm.io/gorm"
)

// News represents a news item from a channel
type News struct {
	ID          uint           `gorm:"primaryKey"`
	ChannelID   string         `gorm:"not null;index:idx_channel_message_id,unique"`
	ChannelName string         `gorm:"not null"`
	MessageID   int            `gorm:"not null;index:idx_channel_message_id,unique"`
	Content     string         `gorm:"type:text"`
	MediaURLs   string         `gorm:"type:text"` // JSON array stored as string
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName returns the table name for News
func (News) TableName() string {
	return "news"
}

// DeliveredNews represents news that has been delivered to a user
type DeliveredNews struct {
	ID          uint           `gorm:"primaryKey"`
	NewsID      uint           `gorm:"not null;index:idx_news_user,unique"`
	UserID      int64          `gorm:"not null;index:idx_news_user,unique;index:idx_user_id"`
	DeliveredAt time.Time      `gorm:"autoCreateTime"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations
	News News `gorm:"foreignKey:NewsID"`
}

// TableName returns the table name for DeliveredNews
func (DeliveredNews) TableName() string {
	return "delivered_news"
}
