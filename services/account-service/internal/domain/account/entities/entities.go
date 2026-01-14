package entities

import (
	"time"

	"github.com/rs/zerolog"
)

// TelegramAccount represents a Telegram account managed by the service
type TelegramAccount struct {
	ID          string    `json:"id"`
	PhoneNumber string    `json:"phoneNumber"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

// InitializationReport contains statistics about account initialization
type InitializationReport struct {
	TotalAccounts      int
	SuccessfulAccounts int
	FailedAccounts     int
	Errors             map[string]error
}

// AccountInitConfig holds configuration for account initialization
type AccountInitConfig struct {
	APIID         int
	APIHash       string
	SessionDir    string
	Accounts      []string
	Logger        zerolog.Logger
	MaxConcurrent int
}
