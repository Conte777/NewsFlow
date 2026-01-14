package telegram

import "time"

// AccountModel represents database model for Telegram account
type AccountModel struct {
	ID              uint       `gorm:"primaryKey"`
	PhoneNumber     string     `gorm:"uniqueIndex;not null;size:32"`
	PhoneHash       string     `gorm:"uniqueIndex;not null;size:64"`
	Status          string     `gorm:"not null;default:'inactive';size:32"`
	LastConnectedAt *time.Time `gorm:""`
	LastError       *string    `gorm:"type:text"`
	CreatedAt       time.Time  `gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime"`
}

// TableName returns the table name for AccountModel
func (AccountModel) TableName() string {
	return "accounts"
}

// SessionModel represents database model for MTProto session
type SessionModel struct {
	ID          uint      `gorm:"primaryKey"`
	AccountID   uint      `gorm:"uniqueIndex;not null"`
	SessionData []byte    `gorm:"type:bytea;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	Account AccountModel `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for SessionModel
func (SessionModel) TableName() string {
	return "sessions"
}

// AccountStatus constants
const (
	AccountStatusInactive = "inactive"
	AccountStatusActive   = "active"
	AccountStatusBanned   = "banned"
	AccountStatusFlood    = "flood_wait"
)
