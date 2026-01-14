package telegram

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/gotd/td/session"
	"gorm.io/gorm"
)

// PostgresSessionStorage implements session.Storage interface using PostgreSQL
type PostgresSessionStorage struct {
	db          *gorm.DB
	phoneNumber string
	phoneHash   string
	accountID   uint
}

// NewPostgresSessionStorage creates a new PostgreSQL-based session storage
func NewPostgresSessionStorage(db *gorm.DB, phoneNumber string) (*PostgresSessionStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if phoneNumber == "" {
		return nil, fmt.Errorf("phone number is required")
	}

	// Generate phone hash (full SHA256 hex string)
	hash := sha256.Sum256([]byte(phoneNumber))
	phoneHash := fmt.Sprintf("%x", hash[:])

	storage := &PostgresSessionStorage{
		db:          db,
		phoneNumber: phoneNumber,
		phoneHash:   phoneHash,
	}

	// Ensure account exists in DB
	if err := storage.ensureAccount(); err != nil {
		return nil, fmt.Errorf("failed to ensure account: %w", err)
	}

	return storage, nil
}

// ensureAccount creates or retrieves the account record in the database
func (s *PostgresSessionStorage) ensureAccount() error {
	var account AccountModel
	result := s.db.Where("phone_hash = ?", s.phoneHash).First(&account)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new account
		account = AccountModel{
			PhoneNumber: s.phoneNumber,
			PhoneHash:   s.phoneHash,
			Status:      AccountStatusInactive,
		}
		if err := s.db.Create(&account).Error; err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}
	} else if result.Error != nil {
		return fmt.Errorf("failed to query account: %w", result.Error)
	}

	s.accountID = account.ID
	return nil
}

// LoadSession loads session data from PostgreSQL
func (s *PostgresSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	var sess SessionModel
	result := s.db.WithContext(ctx).Where("account_id = ?", s.accountID).First(&sess)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, session.ErrNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to load session: %w", result.Error)
	}

	if len(sess.SessionData) == 0 {
		return nil, session.ErrNotFound
	}

	return sess.SessionData, nil
}

// StoreSession stores session data to PostgreSQL
func (s *PostgresSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	var sess SessionModel
	result := s.db.WithContext(ctx).Where("account_id = ?", s.accountID).First(&sess)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new session
		sess = SessionModel{
			AccountID:   s.accountID,
			SessionData: data,
		}
		if err := s.db.WithContext(ctx).Create(&sess).Error; err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	} else if result.Error != nil {
		return fmt.Errorf("failed to query session: %w", result.Error)
	} else {
		// Update existing session
		if err := s.db.WithContext(ctx).Model(&sess).Update("session_data", data).Error; err != nil {
			return fmt.Errorf("failed to update session: %w", err)
		}
	}

	return nil
}

// GetAccountID returns the database account ID
func (s *PostgresSessionStorage) GetAccountID() uint {
	return s.accountID
}

// GetPhoneHash returns the hashed phone number
func (s *PostgresSessionStorage) GetPhoneHash() string {
	return s.phoneHash
}

// UpdateAccountStatus updates the account status in the database
func (s *PostgresSessionStorage) UpdateAccountStatus(ctx context.Context, status string, lastError *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"last_error": lastError,
	}

	if status == AccountStatusActive {
		now := time.Now()
		updates["last_connected_at"] = &now
	}

	return s.db.WithContext(ctx).Model(&AccountModel{}).Where("id = ?", s.accountID).Updates(updates).Error
}

// DeleteSession removes the session from the database
func (s *PostgresSessionStorage) DeleteSession(ctx context.Context) error {
	return s.db.WithContext(ctx).Where("account_id = ?", s.accountID).Delete(&SessionModel{}).Error
}

// SessionExists checks if a session exists in the database
func (s *PostgresSessionStorage) SessionExists(ctx context.Context) bool {
	var count int64
	s.db.WithContext(ctx).Model(&SessionModel{}).Where("account_id = ?", s.accountID).Count(&count)
	return count > 0
}

// Ensure PostgresSessionStorage implements session.Storage interface
var _ session.Storage = (*PostgresSessionStorage)(nil)
