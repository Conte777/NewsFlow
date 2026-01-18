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

func (u *UseCase) GetUserSubscriptions(ctx context.Context, userID int64) ([]entities.SubscriptionView, error) {
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
		userIDs[i] = sub.TelegramID
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

// Saga workflow: Subscription flow

// HandleSubscriptionRequested handles subscription toggle logic:
// - If not subscribed: creates subscription with pending status
// - If already active: initiates unsubscription (toggle behavior)
// - If pending/removing: ignores duplicate request
func (u *UseCase) HandleSubscriptionRequested(ctx context.Context, userID int64, channelID, channelName string) error {
	if userID <= 0 {
		return suberrors.ErrInvalidUserID
	}
	if channelID == "" {
		return suberrors.ErrInvalidChannelID
	}

	// Check if subscription already exists
	existing, err := u.repo.GetByUserAndChannel(ctx, userID, channelID)
	if err == nil && existing != nil {
		// Subscription exists - handle based on status
		switch existing.Status {
		case entities.StatusPending:
			u.logger.Warn().
				Int64("user_id", userID).
				Str("channel_id", channelID).
				Msg("subscription already pending, ignoring duplicate request")
			return nil

		case entities.StatusRemoving:
			u.logger.Warn().
				Int64("user_id", userID).
				Str("channel_id", channelID).
				Msg("unsubscription already in progress, ignoring request")
			return nil

		case entities.StatusActive:
			// Toggle: active subscription -> initiate unsubscription
			u.logger.Info().
				Int64("user_id", userID).
				Str("channel_id", channelID).
				Msg("subscription active, toggling to unsubscription")
			return u.HandleUnsubscriptionRequested(ctx, userID, channelID)
		}
	}

	// No subscription exists - create new with pending status
	sub, err := u.repo.CreateWithStatus(ctx, userID, channelID, channelName, entities.StatusPending)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to create pending subscription")
		return err
	}

	// Send pending event to account-service
	if err := u.kafkaProducer.SendSubscriptionPending(ctx, sub.ID, userID, channelID, channelName); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to send subscription.pending event")
		// Delete the pending subscription since we couldn't notify account-service
		_ = u.repo.Delete(ctx, userID, channelID)
		return err
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Uint("subscription_id", sub.ID).
		Msg("subscription created with pending status, sent to account-service")

	return nil
}

// HandleSubscriptionActivated updates subscription status to active
func (u *UseCase) HandleSubscriptionActivated(ctx context.Context, userID int64, channelID string) error {
	if err := u.repo.UpdateStatus(ctx, userID, channelID, entities.StatusActive); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to activate subscription")
		return err
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("subscription activated successfully")

	return nil
}

// HandleSubscriptionFailed deletes the pending subscription and sends rejection to bot-service
func (u *UseCase) HandleSubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error {
	// Get subscription to retrieve channel name for rejection event
	sub, err := u.repo.GetByUserAndChannel(ctx, userID, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to get subscription for failure handling")
		return err
	}

	channelName := sub.ChannelName

	// Delete the failed subscription
	if err := u.repo.Delete(ctx, userID, channelID); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to delete failed subscription")
		return err
	}

	// Send rejection event to bot-service
	if err := u.kafkaProducer.SendSubscriptionRejected(ctx, userID, channelID, channelName, reason); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to send subscription.rejected event")
		return err
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Str("reason", reason).
		Msg("subscription failed, rejection sent to bot-service")

	return nil
}

// Saga workflow: Unsubscription flow

// HandleUnsubscriptionRequested deletes subscription and sends pending event only if no other subscribers remain
func (u *UseCase) HandleUnsubscriptionRequested(ctx context.Context, userID int64, channelID string) error {
	if userID <= 0 {
		return suberrors.ErrInvalidUserID
	}
	if channelID == "" {
		return suberrors.ErrInvalidChannelID
	}

	// Check if subscription exists and is active
	existing, err := u.repo.GetByUserAndChannel(ctx, userID, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("subscription not found for unsubscription")
		return err
	}

	if existing.Status == entities.StatusPending {
		u.logger.Warn().
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("cannot unsubscribe from pending subscription")
		return suberrors.ErrSubscriptionNotFound
	}

	// Delete subscription from DB immediately
	if err := u.repo.Delete(ctx, userID, channelID); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("failed to delete subscription")
		return err
	}

	// Check if there are remaining subscribers for this channel
	subscribers, err := u.repo.GetByChannelID(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("failed to check remaining subscribers")
		// Subscription already deleted, just log and continue
	}

	// Only send unsubscription.pending if no other subscribers remain
	if len(subscribers) == 0 {
		if err := u.kafkaProducer.SendUnsubscriptionPending(ctx, userID, channelID); err != nil {
			u.logger.Error().Err(err).
				Int64("user_id", userID).
				Str("channel_id", channelID).
				Msg("failed to send unsubscription.pending event")
			return err
		}
		u.logger.Info().
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("last subscriber removed, sent unsubscription.pending to account-service")
	} else {
		u.logger.Info().
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Int("remaining_subscribers", len(subscribers)).
			Msg("subscription deleted, channel still has other subscribers")
	}

	return nil
}

// HandleUnsubscriptionCompleted logs successful unsubscription by account-service
// Note: subscription is already deleted in HandleUnsubscriptionRequested
func (u *UseCase) HandleUnsubscriptionCompleted(ctx context.Context, userID int64, channelID string) error {
	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("unsubscription completed by account-service")

	return nil
}

// HandleUnsubscriptionFailed logs unsubscription failure from account-service
// Note: subscription is already deleted in HandleUnsubscriptionRequested, no rollback possible
// This is called when account-service fails to leave the Telegram channel physically
func (u *UseCase) HandleUnsubscriptionFailed(ctx context.Context, userID int64, channelID, reason string) error {
	u.logger.Warn().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Str("reason", reason).
		Msg("account-service failed to leave channel, but user subscription already deleted")

	return nil
}
