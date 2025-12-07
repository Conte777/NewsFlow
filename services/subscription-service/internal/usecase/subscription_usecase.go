package usecase

import (
	"context"
	"fmt"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"
	"github.com/rs/zerolog"
)

// subscriptionUseCase implements domain.SubscriptionUseCase
type subscriptionUseCase struct {
	repo          domain.SubscriptionRepository
	kafkaProducer domain.KafkaProducer
	logger        zerolog.Logger
}

// NewSubscriptionUseCase creates a new subscription use case
func NewSubscriptionUseCase(
	repo domain.SubscriptionRepository,
	kafkaProducer domain.KafkaProducer,
	logger zerolog.Logger,
) domain.SubscriptionUseCase {
	return &subscriptionUseCase{
		repo:          repo,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

// CreateSubscription creates a new subscription
func (u *subscriptionUseCase) CreateSubscription(ctx context.Context, userID int64, channelID, channelName string) error {
	if userID <= 0 {
		return domain.ErrInvalidUserID
	}

	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	// Check if subscription already exists
	exists, err := u.repo.Exists(ctx, userID, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to check subscription existence")
		return err
	}

	if exists {
		return domain.ErrSubscriptionAlreadyExists
	}

	// Create subscription
	subscription := &domain.Subscription{
		UserID:      userID,
		ChannelID:   channelID,
		ChannelName: channelName,
		IsActive:    true,
	}

	if err := u.repo.Create(ctx, subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to create subscription")
		return err
	}

	// Notify account service
	if err := u.kafkaProducer.NotifyAccountService(ctx, "subscription.created", subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to notify account service")
		// Don't return error here, subscription is already created
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Subscription created successfully")

	return nil
}

// DeleteSubscription deletes a subscription
func (u *subscriptionUseCase) DeleteSubscription(ctx context.Context, userID int64, channelID string) error {
	if userID <= 0 {
		return domain.ErrInvalidUserID
	}

	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	// Delete subscription
	if err := u.repo.Delete(ctx, userID, channelID); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to delete subscription")
		return err
	}

	// Notify account service
	subscription := &domain.Subscription{
		UserID:    userID,
		ChannelID: channelID,
	}

	if err := u.kafkaProducer.NotifyAccountService(ctx, "subscription.deleted", subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to notify account service")
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Subscription deleted successfully")

	return nil
}

// GetUserSubscriptions retrieves all subscriptions for a user
func (u *subscriptionUseCase) GetUserSubscriptions(ctx context.Context, userID int64) ([]domain.Subscription, error) {
	if userID <= 0 {
		return nil, domain.ErrInvalidUserID
	}

	subscriptions, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Msg("Failed to get user subscriptions")
		return nil, err
	}

	return subscriptions, nil
}

// GetChannelSubscribers retrieves all users subscribed to a channel
func (u *subscriptionUseCase) GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error) {
	if channelID == "" {
		return nil, domain.ErrInvalidChannelID
	}

	subscriptions, err := u.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to get channel subscribers")
		return nil, err
	}

	userIDs := make([]int64, len(subscriptions))
	for i, sub := range subscriptions {
		userIDs[i] = sub.UserID
	}

	return userIDs, nil
}

// GetActiveChannels retrieves all active channels
func (u *subscriptionUseCase) GetActiveChannels(ctx context.Context) ([]string, error) {
	channels, err := u.repo.GetActiveChannels(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get active channels")
		return nil, fmt.Errorf("failed to get active channels: %w", err)
	}

	return channels, nil
}
