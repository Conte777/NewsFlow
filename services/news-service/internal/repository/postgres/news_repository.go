package postgres

import (
	"context"
	"errors"

	"github.com/yourusername/telegram-news-feed/news-service/internal/domain"
	"gorm.io/gorm"
)

// newsRepository implements domain.NewsRepository
type newsRepository struct {
	db *gorm.DB
}

// NewNewsRepository creates a new news repository
func NewNewsRepository(db *gorm.DB) domain.NewsRepository {
	return &newsRepository{
		db: db,
	}
}

// Create creates a new news item
func (r *newsRepository) Create(ctx context.Context, news *domain.News) error {
	result := r.db.WithContext(ctx).Create(news)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrNewsAlreadyExists
		}
		return domain.ErrDatabaseOperation
	}
	return nil
}

// GetByID retrieves news by ID
func (r *newsRepository) GetByID(ctx context.Context, id uint) (*domain.News, error) {
	var news domain.News
	result := r.db.WithContext(ctx).First(&news, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNewsNotFound
		}
		return nil, domain.ErrDatabaseOperation
	}
	return &news, nil
}

// GetByChannelAndMessageID retrieves news by channel and message ID
func (r *newsRepository) GetByChannelAndMessageID(ctx context.Context, channelID string, messageID int) (*domain.News, error) {
	var news domain.News
	result := r.db.WithContext(ctx).
		Where("channel_id = ? AND message_id = ?", channelID, messageID).
		First(&news)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNewsNotFound
		}
		return nil, domain.ErrDatabaseOperation
	}

	return &news, nil
}

// Exists checks if news exists
func (r *newsRepository) Exists(ctx context.Context, channelID string, messageID int) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.News{}).
		Where("channel_id = ? AND message_id = ?", channelID, messageID).
		Count(&count)

	if result.Error != nil {
		return false, domain.ErrDatabaseOperation
	}

	return count > 0, nil
}
