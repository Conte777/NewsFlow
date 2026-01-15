package entities

import "time"

type Subscription struct {
	ID          uint      `db:"id" json:"id" gorm:"primaryKey"`
	UserID      int64     `db:"user_id" json:"userId" gorm:"not null;index:idx_user_channel,unique"`
	ChannelID   string    `db:"channel_id" json:"channelId" gorm:"not null;index:idx_user_channel,unique;index:idx_channel"`
	ChannelName string    `db:"channel_name" json:"channelName" gorm:"not null"`
	IsActive    bool      `db:"is_active" json:"isActive" gorm:"not null;default:true"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt" gorm:"autoUpdateTime"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}
