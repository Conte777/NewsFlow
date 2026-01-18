package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/entities"
)

type SubscriptionRepository interface {
	// Legacy methods
	Create(ctx context.Context, telegramUserID int64, channelID, channelName string) error
	Delete(ctx context.Context, telegramUserID int64, channelID string) error
	GetByUserID(ctx context.Context, telegramUserID int64) ([]entities.SubscriptionView, error)
	GetByChannelID(ctx context.Context, channelID string) ([]entities.SubscriptionView, error)
	Exists(ctx context.Context, telegramUserID int64, channelID string) (bool, error)
	GetActiveChannels(ctx context.Context) ([]string, error)

	// Saga workflow methods
	CreateWithStatus(ctx context.Context, telegramUserID int64, channelID, channelName string, status entities.SubscriptionStatus) (*entities.SubscriptionView, error)
	UpdateStatus(ctx context.Context, telegramUserID int64, channelID string, status entities.SubscriptionStatus) error
	GetByUserAndChannel(ctx context.Context, telegramUserID int64, channelID string) (*entities.SubscriptionView, error)
}

type KafkaProducer interface {
	// Saga: Subscription flow
	SendSubscriptionPending(ctx context.Context, subscriptionID uint, telegramUserID int64, channelID, channelName string) error
	SendSubscriptionRejected(ctx context.Context, telegramUserID int64, channelID, channelName, reason string) error

	// Saga: Unsubscription flow
	SendUnsubscriptionPending(ctx context.Context, telegramUserID int64, channelID string) error
	SendUnsubscriptionRejected(ctx context.Context, telegramUserID int64, channelID, channelName, reason string) error

	Close() error
}

type SubscriptionUseCase interface {
	GetUserSubscriptions(ctx context.Context, userID int64) ([]entities.SubscriptionView, error)
	GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error)
	GetActiveChannels(ctx context.Context) ([]string, error)

	// Saga workflow methods
	HandleSubscriptionRequested(ctx context.Context, userID int64, channelID, channelName string) error
	HandleSubscriptionActivated(ctx context.Context, userID int64, channelID string) error
	HandleSubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error

	HandleUnsubscriptionRequested(ctx context.Context, userID int64, channelID string) error
	HandleUnsubscriptionCompleted(ctx context.Context, userID int64, channelID string) error
	HandleUnsubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error
}
