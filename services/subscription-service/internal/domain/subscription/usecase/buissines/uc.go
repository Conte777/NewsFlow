package buissines

import (
	"context"
	"fmt"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/deps"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/entities"
	suberrors "github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/errors"
	"github.com/rs/zerolog"
)

type UseCase struct {
	repo          deps.SubscriptionRepository
	kafkaProducer deps.KafkaProducer
	logger        zerolog.Logger
}

func NewUseCase(
	repo deps.SubscriptionRepository,
	kafkaProducer deps.KafkaProducer,
	logger zerolog.Logger,
) *UseCase {
	return &UseCase{
		repo:          repo,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

func (u *UseCase) CreateSubscription(ctx context.Context, userID int64, channelID, channelName string) error {
	if userID <= 0 {
		return suberrors.ErrInvalidUserID
	}

	if channelID == "" {
		return suberrors.ErrInvalidChannelID
	}

	exists, err := u.repo.Exists(ctx, userID, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to check subscription existence")
		return err
	}

	if exists {
		return suberrors.ErrSubscriptionAlreadyExists
	}

	subscription := &entities.Subscription{
		UserID:      userID,
		ChannelID:   channelID,
		ChannelName: channelName,
		IsActive:    true,
	}

	if err := u.repo.Create(ctx, subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to create subscription")
		return err
	}

	if err := u.kafkaProducer.NotifyAccountService(ctx, "subscription.created", subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to notify account service")
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("subscription created successfully")

	return nil
}

func (u *UseCase) DeleteSubscription(ctx context.Context, userID int64, channelID string) error {
	if userID <= 0 {
		return suberrors.ErrInvalidUserID
	}

	if channelID == "" {
		return suberrors.ErrInvalidChannelID
	}

	if err := u.repo.Delete(ctx, userID, channelID); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to delete subscription")
		return err
	}

	subscription := &entities.Subscription{
		UserID:    userID,
		ChannelID: channelID,
	}

	if err := u.kafkaProducer.NotifyAccountService(ctx, "subscription.deleted", subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to notify account service")
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("subscription deleted successfully")

	return nil
}

func (u *UseCase) GetUserSubscriptions(ctx context.Context, userID int64) ([]entities.Subscription, error) {
	if userID <= 0 {
		return nil, suberrors.ErrInvalidUserID
	}

	subscriptions, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Msg("failed to get user subscriptions")
		return nil, err
	}

	return subscriptions, nil
}

func (u *UseCase) GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error) {
	if channelID == "" {
		return nil, suberrors.ErrInvalidChannelID
	}

	subscriptions, err := u.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("failed to get channel subscribers")
		return nil, err
	}

	userIDs := make([]int64, len(subscriptions))
	for i, sub := range subscriptions {
		userIDs[i] = sub.UserID
	}

	return userIDs, nil
}

func (u *UseCase) GetActiveChannels(ctx context.Context) ([]string, error) {
	channels, err := u.repo.GetActiveChannels(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("failed to get active channels")
		return nil, fmt.Errorf("failed to get active channels: %w", err)
	}

	return channels, nil
}
