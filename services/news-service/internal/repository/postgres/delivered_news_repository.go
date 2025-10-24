package postgres

import (
	"context"
	"errors"

	"github.com/yourusername/telegram-news-feed/news-service/internal/domain"
	"gorm.io/gorm"
)

// deliveredNewsRepository implements domain.DeliveredNewsRepository
type deliveredNewsRepository struct {
	db *gorm.DB
}

// NewDeliveredNewsRepository creates a new delivered news repository
func NewDeliveredNewsRepository(db *gorm.DB) domain.DeliveredNewsRepository {
	return &deliveredNewsRepository{
		db: db,
	}
}

// Create records that news was delivered to user
func (r *deliveredNewsRepository) Create(ctx context.Context, delivered *domain.DeliveredNews) error {
	result := r.db.WithContext(ctx).Create(delivered)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrAlreadyDelivered
		}
		return domain.ErrDatabaseOperation
	}
	return nil
}

// IsDelivered checks if news was already delivered to user
func (r *deliveredNewsRepository) IsDelivered(ctx context.Context, newsID uint, userID int64) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.DeliveredNews{}).
		Where("news_id = ? AND user_id = ?", newsID, userID).
		Count(&count)

	if result.Error != nil {
		return false, domain.ErrDatabaseOperation
	}

	return count > 0, nil
}

// GetUserDeliveredNews retrieves all news delivered to user
func (r *deliveredNewsRepository) GetUserDeliveredNews(ctx context.Context, userID int64, limit int) ([]domain.DeliveredNews, error) {
	var deliveredNews []domain.DeliveredNews
	query := r.db.WithContext(ctx).
		Preload("News").
		Where("user_id = ?", userID).
		Order("delivered_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	result := query.Find(&deliveredNews)
	if result.Error != nil {
		return nil, domain.ErrDatabaseOperation
	}

	return deliveredNews, nil
}
