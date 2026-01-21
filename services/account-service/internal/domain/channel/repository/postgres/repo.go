package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/entities"
	channelerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository implements deps.ChannelRepository using PostgreSQL
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new PostgreSQL channel repository
func NewRepository(db *gorm.DB) deps.ChannelRepository {
	return &Repository{db: db}
}

// AddChannelForAccount adds a channel subscription with account binding
func (r *Repository) AddChannelForAccount(ctx context.Context, phoneNumber, channelID, channelName string, numericID int64) error {
	var account entities.AccountModel
	if err := r.db.WithContext(ctx).Where("phone_number = ?", phoneNumber).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("account not found for phone: %s", phoneNumber)
		}
		return fmt.Errorf("failed to find account: %w", err)
	}

	model := &entities.AccountChannelModel{
		AccountID:   account.ID,
		ChannelID:   channelID,
		ChannelName: channelName,
		NumericID:   numericID,
	}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}, {Name: "channel_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_name", "numeric_id", "updated_at"}),
		}).
		Create(model)
	if result.Error != nil {
		return fmt.Errorf("failed to add channel for account: %w", result.Error)
	}

	return nil
}

// RemoveChannel removes a channel subscription
func (r *Repository) RemoveChannel(ctx context.Context, channelID string) error {
	result := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Delete(&entities.AccountChannelModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to remove channel: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return channelerrors.ErrChannelNotFound
	}

	return nil
}

// RemoveChannelForAccount removes a channel subscription for specific account
func (r *Repository) RemoveChannelForAccount(ctx context.Context, phoneNumber, channelID string) error {
	var account entities.AccountModel
	if err := r.db.WithContext(ctx).Where("phone_number = ?", phoneNumber).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("account not found for phone: %s", phoneNumber)
		}
		return fmt.Errorf("failed to find account: %w", err)
	}

	result := r.db.WithContext(ctx).
		Where("account_id = ? AND channel_id = ?", account.ID, channelID).
		Delete(&entities.AccountChannelModel{})

	if result.Error != nil {
		return fmt.Errorf("failed to remove channel for account: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return channelerrors.ErrChannelNotFound
	}

	return nil
}

// GetAllChannels retrieves all channel subscriptions
func (r *Repository) GetAllChannels(ctx context.Context) ([]entities.ChannelSubscription, error) {
	var models []entities.AccountChannelModel
	if err := r.db.WithContext(ctx).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get all channels: %w", err)
	}

	channels := make([]entities.ChannelSubscription, len(models))
	for i, model := range models {
		channels[i] = *model.ToEntity()
	}

	return channels, nil
}

// GetChannel retrieves a specific channel subscription
func (r *Repository) GetChannel(ctx context.Context, channelID string) (*entities.ChannelSubscription, error) {
	var model entities.AccountChannelModel
	if err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelerrors.ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return model.ToEntity(), nil
}

// GetChannelByNumericID retrieves a channel subscription by its Telegram numeric ID
func (r *Repository) GetChannelByNumericID(ctx context.Context, numericID int64) (*entities.ChannelSubscription, error) {
	var model entities.AccountChannelModel
	if err := r.db.WithContext(ctx).
		Where("numeric_id = ?", numericID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelerrors.ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to get channel by numeric ID: %w", err)
	}

	return model.ToEntity(), nil
}

// ChannelExists checks if a subscription exists for the channel (by any account)
func (r *Repository) ChannelExists(ctx context.Context, channelID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entities.AccountChannelModel{}).
		Where("channel_id = ?", channelID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check channel existence: %w", err)
	}

	return count > 0, nil
}

// UpdateLastProcessedMessageID updates the last processed message ID for a channel
func (r *Repository) UpdateLastProcessedMessageID(ctx context.Context, channelID string, messageID int) error {
	result := r.db.WithContext(ctx).
		Model(&entities.AccountChannelModel{}).
		Where("channel_id = ?", channelID).
		Update("last_processed_message_id", messageID)

	if result.Error != nil {
		return fmt.Errorf("failed to update last processed message ID: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return channelerrors.ErrChannelNotFound
	}

	return nil
}

// GetChannelsByAccount retrieves all active subscriptions for a specific account
func (r *Repository) GetChannelsByAccount(ctx context.Context, phoneNumber string) ([]entities.ChannelSubscription, error) {
	var account entities.AccountModel
	if err := r.db.WithContext(ctx).Where("phone_number = ?", phoneNumber).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("account not found for phone: %s", phoneNumber)
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	var models []entities.AccountChannelModel
	if err := r.db.WithContext(ctx).
		Where("account_id = ?", account.ID).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get channels by account: %w", err)
	}

	channels := make([]entities.ChannelSubscription, len(models))
	for i, model := range models {
		channels[i] = *model.ToEntity()
	}

	return channels, nil
}

// GetAccountPhoneForChannel returns the phone_number of account subscribed to channel
func (r *Repository) GetAccountPhoneForChannel(ctx context.Context, channelID string) (string, error) {
	var model entities.AccountChannelModel
	if err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", channelerrors.ErrChannelNotFound
		}
		return "", fmt.Errorf("failed to get account for channel: %w", err)
	}

	var account entities.AccountModel
	if err := r.db.WithContext(ctx).First(&account, model.AccountID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("account not found for ID: %d", model.AccountID)
		}
		return "", fmt.Errorf("failed to get account: %w", err)
	}

	return account.PhoneNumber, nil
}
