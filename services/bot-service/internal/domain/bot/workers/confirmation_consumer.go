// Package workers contains background workers for the bot domain
package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/deps"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/dto"
)

// ConfirmationConsumer consumes confirmation events from Kafka (Saga workflow)
type ConfirmationConsumer struct {
	reader *kafka.Reader
	sender deps.TelegramSender
	logger zerolog.Logger
	done   chan struct{}
}

// NewConfirmationConsumer creates new Kafka consumer for confirmation events
func NewConfirmationConsumer(cfg *config.KafkaConfig, sender deps.TelegramSender, logger zerolog.Logger) *ConfirmationConsumer {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	topics := []string{
		cfg.TopicSubscriptionConfirmed,
		cfg.TopicUnsubscriptionConfirmed,
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     cfg.GroupID + "-confirmation",
		GroupTopics: topics,
		MinBytes:    1,    // 1 byte - return immediately when message available
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.FirstOffset,
	})

	logger.Info().
		Strs("brokers", brokers).
		Str("group_id", cfg.GroupID+"-confirmation").
		Strs("topics", topics).
		Msg("Kafka confirmation consumer initialized")

	return &ConfirmationConsumer{
		reader: reader,
		sender: sender,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start starts consuming confirmation messages from Kafka
func (c *ConfirmationConsumer) Start(ctx context.Context) {
	c.logger.Info().Msg("Starting Kafka confirmation consumer...")

	go func() {
		for {
			select {
			case <-c.done:
				c.logger.Info().Msg("Kafka confirmation consumer stopped by done signal")
				return
			case <-ctx.Done():
				c.logger.Info().Msg("Kafka confirmation consumer stopped by context cancellation")
				return
			default:
				msg, err := c.reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					c.logger.Error().Err(err).Msg("Failed to read confirmation message from Kafka")
					continue
				}

				c.logger.Debug().
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Received confirmation message from Kafka")

				if err := c.handleConfirmation(ctx, msg.Value); err != nil {
					c.logger.Error().Err(err).Msg("Failed to handle confirmation event")
				}
			}
		}
	}()
}

func (c *ConfirmationConsumer) handleConfirmation(ctx context.Context, data []byte) error {
	var event dto.ConfirmedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal confirmation event: %w", err)
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse user_id: %w", err)
	}

	var message string
	switch event.Type {
	case "subscription_confirmed":
		channelDisplay := event.ChannelName
		if channelDisplay == "" {
			channelDisplay = event.ChannelID
		}
		message = fmt.Sprintf("✅ Вы успешно подписались на канал %s", channelDisplay)

	case "unsubscription_confirmed":
		channelDisplay := event.ChannelName
		if channelDisplay == "" {
			channelDisplay = event.ChannelID
		}
		message = fmt.Sprintf("✅ Вы успешно отписались от канала %s", channelDisplay)

	default:
		c.logger.Warn().Str("type", event.Type).Msg("Unknown confirmation event type")
		return nil
	}

	c.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", event.ChannelID).
		Str("type", event.Type).
		Msg("Sending confirmation notification to user")

	return c.sender.SendMessage(ctx, userID, message)
}

// Stop stops the consumer gracefully
func (c *ConfirmationConsumer) Stop() error {
	c.logger.Info().Msg("Stopping Kafka confirmation consumer...")
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka confirmation reader")
		return err
	}

	c.logger.Info().Msg("Kafka confirmation consumer stopped successfully")
	return nil
}
