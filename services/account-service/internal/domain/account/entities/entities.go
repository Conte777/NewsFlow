package entities

import "time"

// TelegramAccount represents a Telegram account managed by the service
type TelegramAccount struct {
	ID          string    `json:"id"`
	PhoneNumber string    `json:"phoneNumber"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

