package entities

import (
	"fmt"
	"time"
)

// AccountChannelModel is a GORM model for account_channels table
type AccountChannelModel struct {
	ID                     uint      `gorm:"primaryKey"`
	AccountID              uint      `gorm:"not null;uniqueIndex:uq_account_channel"`
	ChannelID              string    `gorm:"not null;size:255;uniqueIndex:uq_account_channel;index"`
	ChannelName            string    `gorm:"size:255;default:''"`
	NumericID              int64     `gorm:"not null;default:0;index"`
	LastProcessedMessageID int       `gorm:"not null;default:0"`
	CreatedAt              time.Time `gorm:"autoCreateTime"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime"`
}

func (AccountChannelModel) TableName() string {
	return "account_channels"
}

// ToEntity converts DB model to domain entity
func (m *AccountChannelModel) ToEntity() *ChannelSubscription {
	return &ChannelSubscription{
		AccountID:              fmt.Sprintf("%d", m.AccountID),
		ChannelID:              m.ChannelID,
		ChannelName:            m.ChannelName,
		NumericID:              m.NumericID,
		LastProcessedMessageID: m.LastProcessedMessageID,
		CreatedAt:              m.CreatedAt,
	}
}

// AccountModel is a GORM model for looking up account_id by phone_number
type AccountModel struct {
	ID          uint   `gorm:"primaryKey"`
	PhoneNumber string `gorm:"uniqueIndex"`
}

func (AccountModel) TableName() string {
	return "accounts"
}
