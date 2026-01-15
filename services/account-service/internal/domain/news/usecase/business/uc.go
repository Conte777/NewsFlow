package business

import (
	"context"
	"errors"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	channeldeps "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/metrics"
	"github.com/rs/zerolog"
)

// UseCase implements news collection business logic
type UseCase struct {
	accountManager domain.AccountManager
	channelRepo    channeldeps.ChannelRepository
	kafkaProducer  domain.KafkaProducer
	logger         zerolog.Logger
	metrics        *metrics.Metrics
}

// NewUseCase creates a new news use case
func NewUseCase(
	accountManager domain.AccountManager,
	channelRepo channeldeps.ChannelRepository,
	kafkaProducer domain.KafkaProducer,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *UseCase {
	return &UseCase{
		accountManager: accountManager,
		channelRepo:    channelRepo,
		kafkaProducer:  kafkaProducer,
		logger:         logger,
		metrics:        m,
	}
}

// CollectNews collects news from all subscribed channels
func (u *UseCase) CollectNews(ctx context.Context) error {
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
