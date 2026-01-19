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

func (r *Repository) findOrCreateUser(ctx context.Context, tx *gorm.DB, telegramID int64) (*entities.User, error) {
	var user entities.User
	result := tx.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&user)
	if result.Error == nil {
		return &user, nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, suberrors.ErrDatabaseOperation
	}

	user = entities.User{TelegramID: telegramID}
	if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, suberrors.ErrDatabaseOperation
	}
	return &user, nil
}

func (r *Repository) findOrCreateChannel(ctx context.Context, tx *gorm.DB, channelID, channelName string) (*entities.Channel, error) {
	var channel entities.Channel
	result := tx.WithContext(ctx).Where("channel_id = ?", channelID).First(&channel)
	if result.Error == nil {
		if channel.ChannelName != channelName {
			channel.ChannelName = channelName
			tx.WithContext(ctx).Save(&channel)
		}
		return &channel, nil
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, suberrors.ErrDatabaseOperation
	}

	channel = entities.Channel{ChannelID: channelID, ChannelName: channelName}
	if err := tx.WithContext(ctx).Create(&channel).Error; err != nil {
		return nil, suberrors.ErrDatabaseOperation
	}
	return &channel, nil
}

func (r *Repository) Create(ctx context.Context, telegramUserID int64, channelID, channelName string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := r.findOrCreateUser(ctx, tx, telegramUserID)
		if err != nil {
			return err
		}

		channel, err := r.findOrCreateChannel(ctx, tx, channelID, channelName)
		if err != nil {
			return err
		}

		subscription := entities.Subscription{
			UserID:    user.ID,
			ChannelID: channel.ID,
			Status:    entities.StatusActive,
		}

		result := tx.Create(&subscription)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return suberrors.ErrSubscriptionAlreadyExists
			}
			return suberrors.ErrDatabaseOperation
		}
		return nil
	})
}

// CreateWithStatus creates a subscription with specified status (for Saga workflow)
func (r *Repository) CreateWithStatus(ctx context.Context, telegramUserID int64, channelID, channelName string, status entities.SubscriptionStatus) (*entities.SubscriptionView, error) {
	var view *entities.SubscriptionView

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := r.findOrCreateUser(ctx, tx, telegramUserID)
		if err != nil {
			return err
		}

		channel, err := r.findOrCreateChannel(ctx, tx, channelID, channelName)
		if err != nil {
			return err
		}

		subscription := entities.Subscription{
			UserID:    user.ID,
			ChannelID: channel.ID,
			Status:    status,
		}

		result := tx.Create(&subscription)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
				return suberrors.ErrSubscriptionAlreadyExists
			}
			return suberrors.ErrDatabaseOperation
		}

		view = &entities.SubscriptionView{
			ID:          subscription.ID,
			TelegramID:  telegramUserID,
			ChannelID:   channelID,
			ChannelName: channelName,
			Status:      subscription.Status,
			CreatedAt:   subscription.CreatedAt,
			UpdatedAt:   subscription.UpdatedAt,
		}

		return nil
	})

	return view, err
}

// UpdateStatus updates the status of a subscription
func (r *Repository) UpdateStatus(ctx context.Context, telegramUserID int64, channelID string, status entities.SubscriptionStatus) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE subscriptions
		SET status = ?, updated_at = NOW()
		WHERE user_id = (SELECT id FROM users WHERE telegram_id = ?)
		AND channel_id = (SELECT id FROM channels WHERE channel_id = ?)
	`, status, telegramUserID, channelID)

	if result.Error != nil {
		return suberrors.ErrDatabaseOperation
	}

	if result.RowsAffected == 0 {
		return suberrors.ErrSubscriptionNotFound
	}

	return nil
}

// GetByUserAndChannel retrieves subscription by user and channel IDs
func (r *Repository) GetByUserAndChannel(ctx context.Context, telegramUserID int64, channelID string) (*entities.SubscriptionView, error) {
	var view entities.SubscriptionView

	result := r.db.WithContext(ctx).Raw(`
		SELECT
			s.id,
			u.telegram_id,
			c.channel_id,
			c.channel_name,
			s.status,
			s.created_at,
			s.updated_at
		FROM subscriptions s
		INNER JOIN users u ON s.user_id = u.id
		INNER JOIN channels c ON s.channel_id = c.id
		WHERE u.telegram_id = ? AND c.channel_id = ?
	`, telegramUserID, channelID).Scan(&view)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	if result.RowsAffected == 0 {
		return nil, suberrors.ErrSubscriptionNotFound
	}

	return &view, nil
}

func (r *Repository) Delete(ctx context.Context, telegramUserID int64, channelID string) error {
	result := r.db.WithContext(ctx).Exec(`
		DELETE FROM subscriptions
		WHERE user_id = (SELECT id FROM users WHERE telegram_id = ?)
		AND channel_id = (SELECT id FROM channels WHERE channel_id = ?)
	`, telegramUserID, channelID)

	if result.Error != nil {
		return suberrors.ErrDatabaseOperation
	}

	if result.RowsAffected == 0 {
		return suberrors.ErrSubscriptionNotFound
	}

	return nil
}

func (r *Repository) GetByUserID(ctx context.Context, telegramUserID int64) ([]entities.SubscriptionView, error) {
	var views []entities.SubscriptionView

	result := r.db.WithContext(ctx).Raw(`
		SELECT
			s.id,
			u.telegram_id,
			c.channel_id,
			c.channel_name,
			s.status,
			s.created_at,
			s.updated_at
		FROM subscriptions s
		INNER JOIN users u ON s.user_id = u.id
		INNER JOIN channels c ON s.channel_id = c.id
		WHERE u.telegram_id = ? AND s.status = 'active'
		ORDER BY s.created_at DESC
	`, telegramUserID).Scan(&views)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	return views, nil
}

func (r *Repository) GetByChannelID(ctx context.Context, channelID string) ([]entities.SubscriptionView, error) {
	var views []entities.SubscriptionView

	result := r.db.WithContext(ctx).Raw(`
		SELECT
			s.id,
			u.telegram_id,
			c.channel_id,
			c.channel_name,
			s.status,
			s.created_at,
			s.updated_at
		FROM subscriptions s
		INNER JOIN users u ON s.user_id = u.id
		INNER JOIN channels c ON s.channel_id = c.id
		WHERE c.channel_id = ? AND s.status = 'active'
	`, channelID).Scan(&views)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	return views, nil
}

func (r *Repository) Exists(ctx context.Context, telegramUserID int64, channelID string) (bool, error) {
	var count int64

	result := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM subscriptions s
		INNER JOIN users u ON s.user_id = u.id
		INNER JOIN channels c ON s.channel_id = c.id
		WHERE u.telegram_id = ? AND c.channel_id = ? AND s.status = 'active'
	`, telegramUserID, channelID).Scan(&count)

	if result.Error != nil {
		return false, suberrors.ErrDatabaseOperation
	}

	return count > 0, nil
}

func (r *Repository) GetActiveChannels(ctx context.Context) ([]string, error) {
	var channels []string

	result := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT c.channel_id
		FROM subscriptions s
		INNER JOIN channels c ON s.channel_id = c.id
		WHERE s.status = 'active'
	`).Scan(&channels)

	if result.Error != nil {
		return nil, suberrors.ErrDatabaseOperation
	}

	return channels, nil
}

func (r *Repository) DeleteOrphanedChannel(ctx context.Context, channelID string) error {
	result := r.db.WithContext(ctx).Exec(`
		DELETE FROM channels
		WHERE channel_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM subscriptions s
			INNER JOIN channels c ON s.channel_id = c.id
			WHERE c.channel_id = ?
		)
	`, channelID, channelID)

	if result.Error != nil {
		return suberrors.ErrDatabaseOperation
	}

	return nil
}
