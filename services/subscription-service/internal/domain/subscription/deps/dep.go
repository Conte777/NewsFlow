package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/entities"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *entities.Subscription) error
	Delete(ctx context.Context, userID int64, channelID string) error
	GetByUserID(ctx context.Context, userID int64) ([]entities.Subscription, error)
	GetByChannelID(ctx context.Context, channelID string) ([]entities.Subscription, error)
	Exists(ctx context.Context, userID int64, channelID string) (bool, error)
	GetActiveChannels(ctx context.Context) ([]string, error)
}

type KafkaProducer interface {
	NotifyAccountService(ctx context.Context, eventType string, subscription *entities.Subscription) error
	Close() error
}
