package postgres

import (
	"context"
	"errors"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/deps"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/entities"
	suberrors "github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/errors"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) deps.SubscriptionRepository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, subscription *entities.Subscription) error {
	result := r.db.WithContext(ctx).Create(subscription)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return suberrors.ErrSubscriptionAlreadyExists
		}
		return suberrors.ErrDatabaseOperation
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, userID int64, channelID string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		Delete(&entities.Subscription{})

	if result.Error != nil {
		return suberrors.ErrDatabaseOperation
	}

	if result.RowsAffected == 0 {
		return suberrors.ErrSubscriptionNotFound
	}

	return nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID int64) ([]entities.Subscription, error) {
	var subscriptions []entities.Subscription
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").
		Find(&subscriptions)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	return subscriptions, nil
}

func (r *Repository) GetByChannelID(ctx context.Context, channelID string) ([]entities.Subscription, error) {
	var subscriptions []entities.Subscription
	result := r.db.WithContext(ctx).
		Where("channel_id = ? AND is_active = ?", channelID, true).
		Find(&subscriptions)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	return subscriptions, nil
}

func (r *Repository) Exists(ctx context.Context, userID int64, channelID string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&entities.Subscription{}).
		Where("user_id = ? AND channel_id = ? AND is_active = ?", userID, channelID, true).
		Count(&count)

	if result.Error != nil {
		return false, suberrors.ErrDatabaseOperation
	}

	return count > 0, nil
}

func (r *Repository) GetActiveChannels(ctx context.Context) ([]string, error) {
	var channels []string
	result := r.db.WithContext(ctx).
		Model(&entities.Subscription{}).
		Where("is_active = ?", true).
		Distinct("channel_id").
		Pluck("channel_id", &channels)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	return channels, nil
}
