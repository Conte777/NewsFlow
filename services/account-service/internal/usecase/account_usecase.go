package usecase

import (
	"context"
	"fmt"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/rs/zerolog"
)

// accountUseCase implements domain.AccountUseCase
type accountUseCase struct {
	accountManager domain.AccountManager
	channelRepo    domain.ChannelRepository
	kafkaProducer  domain.KafkaProducer
	logger         zerolog.Logger
}

// NewAccountUseCase creates a new account use case
func NewAccountUseCase(
	accountManager domain.AccountManager,
	channelRepo domain.ChannelRepository,
	kafkaProducer domain.KafkaProducer,
	logger zerolog.Logger,
) domain.AccountUseCase {
	return &accountUseCase{
		accountManager: accountManager,
		channelRepo:    channelRepo,
		kafkaProducer:  kafkaProducer,
		logger:         logger,
	}
}

// SubscribeToChannel subscribes an account to a channel
func (u *accountUseCase) SubscribeToChannel(ctx context.Context, channelID, channelName string) error {
	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	// Check if already subscribed
	exists, err := u.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to check channel existence")
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
		return domain.ErrNoActiveAccounts
	}

	// Join channel
	if err := client.JoinChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to join channel")
		return domain.ErrSubscriptionFailed
	}

	// Get channel name if not provided
	if channelName == "" {
		channelName, err = client.GetChannelInfo(ctx, channelID)
		if err != nil {
			u.logger.Warn().Err(err).
				Str("channel_id", channelID).
				Msg("Failed to get channel name, using channel ID")
			channelName = channelID
		}
	}

	// Save to repository
	if err := u.channelRepo.AddChannel(ctx, channelID, channelName); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to save channel subscription")
		return err
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Msg("Successfully subscribed to channel")

	return nil
}

// UnsubscribeFromChannel unsubscribes an account from a channel
func (u *accountUseCase) UnsubscribeFromChannel(ctx context.Context, channelID string) error {
	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	// Check if subscribed
	exists, err := u.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to check channel existence")
		return err
	}

	if !exists {
		u.logger.Debug().
			Str("channel_id", channelID).
			Msg("Not subscribed to channel")
		return domain.ErrChannelNotFound
	}

	// Get available account
	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get available account")
		return domain.ErrNoActiveAccounts
	}

	// Leave channel
	if err := client.LeaveChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to leave channel")
		return domain.ErrUnsubscriptionFailed
	}

	// Remove from repository
	if err := u.channelRepo.RemoveChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to remove channel subscription")
		return err
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Msg("Successfully unsubscribed from channel")

	return nil
}

// CollectNews collects news from all subscribed channels
func (u *accountUseCase) CollectNews(ctx context.Context) error {
	// Get all subscribed channels
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get subscribed channels")
		return err
	}

	if len(channels) == 0 {
		u.logger.Debug().Msg("No subscribed channels")
		return nil
	}

	u.logger.Info().Int("channels_count", len(channels)).Msg("Collecting news from channels")

	// Get available account
	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get available account")
		return domain.ErrNoActiveAccounts
	}

	// Collect news from each channel
	for _, channel := range channels {
		newsItems, err := client.GetChannelMessages(ctx, channel.ChannelID, 10, 0)
		if err != nil {
			u.logger.Error().Err(err).
				Str("channel_id", channel.ChannelID).
				Msg("Failed to get channel messages")
			continue
		}

		// Send each news item to Kafka
		for _, news := range newsItems {
			if err := u.kafkaProducer.SendNewsReceived(ctx, &news); err != nil {
				u.logger.Error().Err(err).
					Str("channel_id", news.ChannelID).
					Int("message_id", news.MessageID).
					Msg("Failed to send news to Kafka")
				continue
			}

			u.logger.Debug().
				Str("channel_id", news.ChannelID).
				Int("message_id", news.MessageID).
				Msg("News sent to Kafka")
		}
	}

	return nil
}

// GetActiveChannels returns all channels that are subscribed
func (u *accountUseCase) GetActiveChannels(ctx context.Context) ([]string, error) {
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get active channels")
		return nil, fmt.Errorf("failed to get active channels: %w", err)
	}

	channelIDs := make([]string, len(channels))
	for i, channel := range channels {
		channelIDs[i] = channel.ChannelID
	}

	return channelIDs, nil
}
