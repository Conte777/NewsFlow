package business

import (
	"context"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	channelerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/errors"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/metrics"
	"github.com/rs/zerolog"
)

// UseCase implements channel business logic
type UseCase struct {
	accountManager domain.AccountManager
	channelRepo    deps.ChannelRepository
	logger         zerolog.Logger
	metrics        *metrics.Metrics
}

// NewUseCase creates a new channel use case
func NewUseCase(
	accountManager domain.AccountManager,
	channelRepo deps.ChannelRepository,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *UseCase {
	return &UseCase{
		accountManager: accountManager,
		channelRepo:    channelRepo,
		logger:         logger,
		metrics:        m,
	}
}

// Subscribe subscribes to a channel
func (u *UseCase) Subscribe(ctx context.Context, channelID, channelName string) error {
	start := time.Now()

	if channelID == "" {
		return channelerrors.ErrInvalidChannelID
	}

	// Check if already subscribed
	exists, err := u.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to check channel existence")
		u.metrics.RecordSubscriptionError("repository_error")
		return err
	}

	if exists {
		u.logger.Debug().
			Str("channel_id", channelID).
			Msg("Already subscribed to channel")
		return nil
	}

	// Get available account
	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get available account")
		u.metrics.RecordSubscriptionError("no_active_accounts")
		return domain.ErrNoActiveAccounts
	}

	// Join channel
	if err := client.JoinChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to join channel")
		u.metrics.RecordSubscriptionError("join_failed")
		return channelerrors.ErrSubscriptionFailed
	}

	// Get channel name if not provided
	if channelName == "" {
		info, err := client.GetChannelInfo(ctx, channelID)
		if err != nil {
			u.logger.Warn().Err(err).
				Str("channel_id", channelID).
				Msg("Failed to get channel info, using channel ID")
			channelName = channelID
		} else {
			channelName = info.Title
		}
	}

	// Save to repository with account binding
	phoneNumber := client.GetAccountID() // Returns phone_number
	if err := u.channelRepo.AddChannelForAccount(ctx, phoneNumber, channelID, channelName); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Str("phone_number", phoneNumber).
			Msg("Failed to save channel subscription")
		u.metrics.RecordSubscriptionError("repository_save_failed")
		return err
	}

	// Record successful subscription
	duration := time.Since(start).Seconds()
	u.metrics.RecordSubscription(duration)

	// Update active subscriptions count
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err == nil {
		u.metrics.UpdateActiveSubscriptions(len(channels))
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Msg("Successfully subscribed to channel")

	return nil
}

// Unsubscribe unsubscribes from a channel using the account that is subscribed to it
func (u *UseCase) Unsubscribe(ctx context.Context, channelID string) error {
	start := time.Now()

	if channelID == "" {
		return channelerrors.ErrInvalidChannelID
	}

	// Check if subscribed
	exists, err := u.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to check channel existence")
		u.metrics.RecordUnsubscriptionError("repository_error")
		return err
	}

	if !exists {
		u.logger.Debug().
			Str("channel_id", channelID).
			Msg("Not subscribed to channel")
		return channelerrors.ErrChannelNotFound
	}

	// Get phone_number of the account that subscribed to this channel
	phoneNumber, err := u.channelRepo.GetAccountPhoneForChannel(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to get account for channel")
		u.metrics.RecordUnsubscriptionError("account_lookup_failed")
		return err
	}

	// Get the specific client by phone number
	client, err := u.accountManager.GetAccountByPhone(phoneNumber)
	if err != nil {
		u.logger.Error().Err(err).
			Str("phone_number", phoneNumber).
			Msg("Failed to get account by phone")
		u.metrics.RecordUnsubscriptionError("account_not_found")
		return domain.ErrNoActiveAccounts
	}

	// Leave channel using the correct account
	if err := client.LeaveChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Str("phone_number", phoneNumber).
			Msg("Failed to leave channel")
		u.metrics.RecordUnsubscriptionError("leave_failed")
		return channelerrors.ErrUnsubscriptionFailed
	}

	// Remove from repository
	if err := u.channelRepo.RemoveChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to remove channel subscription")
		u.metrics.RecordUnsubscriptionError("repository_remove_failed")
		return err
	}

	// Record successful unsubscription
	duration := time.Since(start).Seconds()
	u.metrics.RecordUnsubscription(duration)

	// Update active subscriptions count
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err == nil {
		u.metrics.UpdateActiveSubscriptions(len(channels))
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Str("phone_number", phoneNumber).
		Msg("Successfully unsubscribed from channel")

	return nil
}

// GetActiveChannels returns all active channel IDs
func (u *UseCase) GetActiveChannels(ctx context.Context) ([]string, error) {
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get active channels")
		return nil, err
	}

	channelIDs := make([]string, len(channels))
	for i, channel := range channels {
		channelIDs[i] = channel.ChannelID
	}

	return channelIDs, nil
}
