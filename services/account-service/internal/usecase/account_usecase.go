package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/metrics"
	"github.com/rs/zerolog"
)

// accountUseCase implements domain.AccountUseCase
type accountUseCase struct {
	accountManager domain.AccountManager
	channelRepo    domain.ChannelRepository
	kafkaProducer  domain.KafkaProducer
	logger         zerolog.Logger
	metrics        *metrics.Metrics
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
		metrics:        metrics.DefaultMetrics,
	}
}

// SubscribeToChannel subscribes an account to a channel
func (u *accountUseCase) SubscribeToChannel(ctx context.Context, channelID, channelName string) error {
	start := time.Now()

	if channelID == "" {
		return domain.ErrInvalidChannelID
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
		return domain.ErrSubscriptionFailed
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

	// Save to repository
	if err := u.channelRepo.AddChannel(ctx, channelID, channelName); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
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

// UnsubscribeFromChannel unsubscribes an account from a channel
func (u *accountUseCase) UnsubscribeFromChannel(ctx context.Context, channelID string) error {
	start := time.Now()

	if channelID == "" {
		return domain.ErrInvalidChannelID
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
		return domain.ErrChannelNotFound
	}

	// Get available account
	client, err := u.accountManager.GetAvailableAccount()
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get available account")
		u.metrics.RecordUnsubscriptionError("no_active_accounts")
		return domain.ErrNoActiveAccounts
	}

	// Leave channel
	if err := client.LeaveChannel(ctx, channelID); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to leave channel")
		u.metrics.RecordUnsubscriptionError("leave_failed")
		return domain.ErrUnsubscriptionFailed
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
		Msg("Successfully unsubscribed from channel")

	return nil
}

// CollectNews collects news from all subscribed channels
func (u *accountUseCase) CollectNews(ctx context.Context) error {
	start := time.Now()

	// Get all subscribed channels
	channels, err := u.channelRepo.GetAllChannels(ctx)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to get subscribed channels")
		u.metrics.RecordNewsCollectionError()
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
		u.metrics.RecordNewsCollectionError()
		return domain.ErrNoActiveAccounts
	}

	// Track collected news statistics
	totalCollected := 0
	totalSent := 0
	totalSkipped := 0

	// Rate limiter to avoid Telegram API flood (100ms between requests)
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	// Collect news from each channel
	for i, channel := range channels {
		// Apply rate limiting (skip for first channel)
		if i > 0 {
			select {
			case <-ctx.Done():
				u.logger.Warn().
					Int("processed_channels", i).
					Int("total_channels", len(channels)).
					Msg("Collection cancelled by context")
				return ctx.Err()
			case <-rateLimiter.C:
				// Continue with next channel
			}
		}

		newsItems, err := client.GetChannelMessages(ctx, channel.ChannelID, 10, 0)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				u.logger.Warn().
					Str("channel_id", channel.ChannelID).
					Msg("Timeout getting channel messages")
			} else {
				u.logger.Error().Err(err).
					Str("channel_id", channel.ChannelID).
					Msg("Failed to get channel messages")
			}
			continue
		}

		totalCollected += len(newsItems)

		// Track the highest message ID for this channel
		maxMessageID := channel.LastProcessedMessageID

		// Send each news item to Kafka (filter duplicates)
		for _, news := range newsItems {
			// Skip already processed messages
			if news.MessageID <= channel.LastProcessedMessageID {
				totalSkipped++
				u.logger.Debug().
					Str("channel_id", news.ChannelID).
					Int("message_id", news.MessageID).
					Int("last_processed", channel.LastProcessedMessageID).
					Msg("Skipping already processed message")
				continue
			}

			kafkaStart := time.Now()
			if err := u.kafkaProducer.SendNewsReceived(ctx, &news); err != nil {
				u.logger.Error().Err(err).
					Str("channel_id", news.ChannelID).
					Int("message_id", news.MessageID).
					Msg("Failed to send news to Kafka")
				u.metrics.RecordKafkaError("send_failed")
				continue
			}

			// Record Kafka message sent with duration
			kafkaDuration := time.Since(kafkaStart).Seconds()
			u.metrics.RecordKafkaMessage(kafkaDuration)
			totalSent++

			// Track the highest message ID
			if news.MessageID > maxMessageID {
				maxMessageID = news.MessageID
			}

			u.logger.Debug().
				Str("channel_id", news.ChannelID).
				Int("message_id", news.MessageID).
				Msg("News sent to Kafka")
		}

		// Update LastProcessedMessageID if we processed any new messages
		if maxMessageID > channel.LastProcessedMessageID {
			if err := u.channelRepo.UpdateLastProcessedMessageID(ctx, channel.ChannelID, maxMessageID); err != nil {
				u.logger.Error().Err(err).
					Str("channel_id", channel.ChannelID).
					Int("message_id", maxMessageID).
					Msg("Failed to update last processed message ID")
			} else {
				u.logger.Debug().
					Str("channel_id", channel.ChannelID).
					Int("last_processed_message_id", maxMessageID).
					Msg("Updated last processed message ID")
			}
		}
	}

	// Log collection results
	u.logger.Info().
		Int("total_collected", totalCollected).
		Int("total_sent", totalSent).
		Int("total_skipped", totalSkipped).
		Int("channels_count", len(channels)).
		Msg("News collection completed")

	// Record news collection metrics
	duration := time.Since(start).Seconds()
	u.metrics.RecordNewsCollection(totalSent, duration)

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
