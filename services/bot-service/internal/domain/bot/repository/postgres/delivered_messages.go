package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/deps"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
)

type deliveredMessageRepository struct {
	db *gorm.DB
}

// NewDeliveredMessageRepository creates a new delivered message repository
func NewDeliveredMessageRepository(db *gorm.DB) deps.DeliveredMessageRepository {
	return &deliveredMessageRepository{db: db}
}

// Save saves a delivered message record
func (r *deliveredMessageRepository) Save(ctx context.Context, msg *entities.DeliveredMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// GetByNewsIDAndUserID retrieves a delivered message by news ID and user ID
func (r *deliveredMessageRepository) GetByNewsIDAndUserID(ctx context.Context, newsID uint, userID int64) (*entities.DeliveredMessage, error) {
	var msg entities.DeliveredMessage
	err := r.db.WithContext(ctx).
		Where("news_id = ? AND user_id = ?", newsID, userID).
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetByNewsID retrieves all delivered messages for a news item
func (r *deliveredMessageRepository) GetByNewsID(ctx context.Context, newsID uint) ([]entities.DeliveredMessage, error) {
	var msgs []entities.DeliveredMessage
	err := r.db.WithContext(ctx).
		Where("news_id = ?", newsID).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// Delete deletes a delivered message record
func (r *deliveredMessageRepository) Delete(ctx context.Context, newsID uint, userID int64) error {
	return r.db.WithContext(ctx).
		Where("news_id = ? AND user_id = ?", newsID, userID).
		Delete(&entities.DeliveredMessage{}).Error
}
