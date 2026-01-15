package telegram

import (
	"context"

	"gorm.io/gorm"
)

// AccountRepository provides access to account data in the database
type AccountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new AccountRepository
func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// GetEnabledPhoneNumbers returns phone numbers of all enabled accounts
func (r *AccountRepository) GetEnabledPhoneNumbers(ctx context.Context) ([]string, error) {
	var accounts []AccountModel
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Select("phone_number").
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	phoneNumbers := make([]string, 0, len(accounts))
	for _, acc := range accounts {
		phoneNumbers = append(phoneNumbers, acc.PhoneNumber)
	}

	return phoneNumbers, nil
}

// SetAccountEnabled enables or disables an account by phone number
func (r *AccountRepository) SetAccountEnabled(ctx context.Context, phoneNumber string, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&AccountModel{}).
		Where("phone_number = ?", phoneNumber).
		Update("enabled", enabled).Error
}
