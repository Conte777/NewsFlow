package postgres

import (
	"context"
	"errors"

	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/deps"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/entities"
	domainerrors "github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/errors"
	"gorm.io/gorm"
)

type newsRepository struct {
	db *gorm.DB
}

// NewNewsRepository creates a new news repository
func NewNewsRepository(db *gorm.DB) deps.NewsRepository {
	return &newsRepository{
		db: db,
	}
}

// Create creates a new news item
func (r *newsRepository) Create(ctx context.Context, news *entities.News) error {
	result := r.db.WithContext(ctx).Create(news)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domainerrors.ErrNewsAlreadyExists
		}
		return domainerrors.ErrDatabaseOperation
	}
	return nil
}

// GetByID retrieves news by ID
func (r *newsRepository) GetByID(ctx context.Context, id uint) (*entities.News, error) {
	var news entities.News
	result := r.db.WithContext(ctx).First(&news, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNewsNotFound
		}
		return nil, domainerrors.ErrDatabaseOperation
	}
	return &news, nil
}

// GetByChannelAndMessageID retrieves news by channel and message ID
func (r *newsRepository) GetByChannelAndMessageID(ctx context.Context, channelID string, messageID int) (*entities.News, error) {
	var news entities.News
	result := r.db.WithContext(ctx).
		Where("channel_id = ? AND message_id = ?", channelID, messageID).
		First(&news)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNewsNotFound
		}
		return nil, domainerrors.ErrDatabaseOperation
	}

	return &news, nil
}

// Exists checks if news exists
func (r *newsRepository) Exists(ctx context.Context, channelID string, messageID int) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&entities.News{}).
		Where("channel_id = ? AND message_id = ?", channelID, messageID).
		Count(&count)

	if result.Error != nil {
		return false, domainerrors.ErrDatabaseOperation
	}

	return count > 0, nil
}

// Update updates an existing news item
func (r *newsRepository) Update(ctx context.Context, news *entities.News) error {
	result := r.db.WithContext(ctx).Save(news)
	if result.Error != nil {
		return domainerrors.ErrDatabaseOperation
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNewsNotFound
	}
	return nil
}

// SoftDelete performs soft delete on a single news item
func (r *newsRepository) SoftDelete(ctx context.Context, channelID string, messageID int) error {
	result := r.db.WithContext(ctx).
		Where("channel_id = ? AND message_id = ?", channelID, messageID).
		Delete(&entities.News{})

	if result.Error != nil {
		return domainerrors.ErrDatabaseOperation
	}
	return nil
}

// SoftDeleteBatch performs soft delete on multiple news items, returns IDs of deleted news
func (r *newsRepository) SoftDeleteBatch(ctx context.Context, channelID string, messageIDs []int) ([]uint, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	// First, get the IDs of news items that will be deleted
	var newsItems []entities.News
	result := r.db.WithContext(ctx).
		Select("id").
		Where("channel_id = ? AND message_id IN ?", channelID, messageIDs).
		Find(&newsItems)

	if result.Error != nil {
		return nil, domainerrors.ErrDatabaseOperation
	}

	if len(newsItems) == 0 {
		return nil, nil
	}

	// Collect IDs
	ids := make([]uint, 0, len(newsItems))
	for _, n := range newsItems {
		ids = append(ids, n.ID)
	}

	// Perform soft delete
	result = r.db.WithContext(ctx).
		Where("channel_id = ? AND message_id IN ?", channelID, messageIDs).
		Delete(&entities.News{})

	if result.Error != nil {
		return nil, domainerrors.ErrDatabaseOperation
	}

	return ids, nil
}

// deliveredNewsRepository implements deps.DeliveredNewsRepository
type deliveredNewsRepository struct {
	db *gorm.DB
}

// NewDeliveredNewsRepository creates a new delivered news repository
func NewDeliveredNewsRepository(db *gorm.DB) deps.DeliveredNewsRepository {
	return &deliveredNewsRepository{
		db: db,
	}
}

// Create records that news was delivered to user
func (r *deliveredNewsRepository) Create(ctx context.Context, delivered *entities.DeliveredNews) error {
	result := r.db.WithContext(ctx).Create(delivered)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domainerrors.ErrAlreadyDelivered
		}
		return domainerrors.ErrDatabaseOperation
	}
	return nil
}

// IsDelivered checks if news was already delivered to user
func (r *deliveredNewsRepository) IsDelivered(ctx context.Context, newsID uint, userID int64) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&entities.DeliveredNews{}).
		Where("news_id = ? AND user_id = ?", newsID, userID).
		Count(&count)

	if result.Error != nil {
		return false, domainerrors.ErrDatabaseOperation
	}

	return count > 0, nil
}

// GetUserDeliveredNews retrieves all news delivered to user
func (r *deliveredNewsRepository) GetUserDeliveredNews(ctx context.Context, userID int64, limit int) ([]entities.DeliveredNews, error) {
	var deliveredNews []entities.DeliveredNews
	query := r.db.WithContext(ctx).
		Preload("News").
		Where("user_id = ?", userID).
		Order("delivered_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	result := query.Find(&deliveredNews)
	if result.Error != nil {
		return nil, domainerrors.ErrDatabaseOperation
	}

	return deliveredNews, nil
}

// GetUsersByChannelID returns distinct users who received news from channel (for fallback)
func (r *deliveredNewsRepository) GetUsersByChannelID(ctx context.Context, channelID string) ([]int64, error) {
	var userIDs []int64
	result := r.db.WithContext(ctx).
		Model(&entities.DeliveredNews{}).
		Select("DISTINCT delivered_news.user_id").
		Joins("JOIN news ON delivered_news.news_id = news.id").
		Where("news.channel_id = ?", channelID).
		Pluck("user_id", &userIDs)

	if result.Error != nil {
		return nil, domainerrors.ErrDatabaseOperation
	}

	return userIDs, nil
}

// GetUsersByNewsID returns all user IDs who received the specific news
func (r *deliveredNewsRepository) GetUsersByNewsID(ctx context.Context, newsID uint) ([]int64, error) {
	var userIDs []int64
	result := r.db.WithContext(ctx).
		Model(&entities.DeliveredNews{}).
		Where("news_id = ?", newsID).
		Pluck("user_id", &userIDs)

	if result.Error != nil {
		return nil, domainerrors.ErrDatabaseOperation
	}

	return userIDs, nil
}
