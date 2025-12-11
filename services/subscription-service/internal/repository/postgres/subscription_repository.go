package postgres

import (
	"context"
	"errors"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
	"gorm.io/gorm"
)

// subscriptionRepository implements domain.SubscriptionRepository
type subscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *gorm.DB) domain.SubscriptionRepository {
	return &subscriptionRepository{
		db: db,
	}
}

// Create creates a new subscription
func (r *subscriptionRepository) Create(ctx context.Context, subscription *domain.Subscription) error {
	result := r.db.WithContext(ctx).Create(subscription)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrSubscriptionAlreadyExists
		}
		return domain.ErrDatabaseOperation
	}
	return nil
}

// Delete deletes a subscription
func (r *subscriptionRepository) Delete(ctx context.Context, userID int64, channelID string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		Delete(&domain.Subscription{})

	if result.Error != nil {
		return domain.ErrDatabaseOperation
	}

	if result.RowsAffected == 0 {
		return domain.ErrSubscriptionNotFound
	}

	return nil
}

// GetByUserID retrieves all subscriptions for a user
func (r *subscriptionRepository) GetByUserID(ctx context.Context, userID int64) ([]domain.Subscription, error) {
	var subscriptions []domain.Subscription
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").
		Find(&subscriptions)

	if result.Error != nil {
		return nil, domain.ErrDatabaseOperation
	}

	return subscriptions, nil
}

// GetByChannelID retrieves all subscriptions for a channel
func (r *subscriptionRepository) GetByChannelID(ctx context.Context, channelID string) ([]domain.Subscription, error) {
	var subscriptions []domain.Subscription
	result := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_active = ?", channelID, true).
		Find(&subscriptions)

	if result.Error != nil {
		return nil, domain.ErrDatabaseOperation
	}

	return subscriptions, nil
}

// Exists checks if subscription exists
func (r *subscriptionRepository) Exists(ctx context.Context, userID int64, channelID string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.Subscription{}).
		Where("user_id = ? AND channel_id = ? AND is_active = ?", userID, channelID, true).
		Count(&count)

	if result.Error != nil {
		return false, domain.ErrDatabaseOperation
	}

	return count > 0, nil
}

// GetActiveChannels retrieves all active channels
func (r *subscriptionRepository) GetActiveChannels(ctx context.Context) ([]string, error) {
	var channels []string
	result := r.db.WithContext(ctx).
		Model(&domain.Subscription{}).
		Where("is_active = ?", true).
		Distinct("channel_id").
		Pluck("channel_id", &channels)

	if result.Error != nil {
		return nil, domain.ErrDatabaseOperation
	}

	return channels, nil
}
