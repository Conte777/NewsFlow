package entities

import (
	"time"

	"gorm.io/gorm"
)

// News represents a news item from a channel
type News struct {
	ID            uint           `gorm:"primaryKey" db:"id" json:"id"`
	ChannelID     string         `gorm:"not null;index:idx_channel_message_id,unique" db:"channel_id" json:"channelId"`
	ChannelName   string         `gorm:"not null" db:"channel_name" json:"channelName"`
	MessageID     int            `gorm:"not null;index:idx_channel_message_id,unique" db:"message_id" json:"messageId"`
	Content       string         `gorm:"type:text" db:"content" json:"content"`
	MediaURLs     string         `gorm:"type:text" db:"media_urls" json:"mediaUrls"`
	MediaMetadata string         `gorm:"type:text" db:"media_metadata" json:"mediaMetadata,omitempty"` // JSON-encoded media metadata
	CreatedAt     time.Time      `gorm:"autoCreateTime" db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" db:"updated_at" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" db:"deleted_at" json:"deletedAt,omitempty"`
}

// TableName returns the table name for News
func (News) TableName() string {
	return "news"
}

// DeliveredNews represents news that has been delivered to a user
type DeliveredNews struct {
	ID          uint           `gorm:"primaryKey" db:"id" json:"id"`
	NewsID      uint           `gorm:"not null;index:idx_news_user,unique" db:"news_id" json:"newsId"`
	UserID      int64          `gorm:"not null;index:idx_news_user,unique;index:idx_user_id" db:"user_id" json:"userId"`
	DeliveredAt time.Time      `gorm:"autoCreateTime" db:"delivered_at" json:"deliveredAt"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" db:"updated_at" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" db:"deleted_at" json:"deletedAt,omitempty"`

	// Relations
	News News `gorm:"foreignKey:NewsID" json:"news,omitempty"`
}

// TableName returns the table name for DeliveredNews
func (DeliveredNews) TableName() string {
	return "delivered_news"
}
