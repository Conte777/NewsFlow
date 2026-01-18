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

// RejectionConsumer consumes rejection events from Kafka (Saga workflow)
type RejectionConsumer struct {
	reader *kafka.Reader
	sender deps.TelegramSender
	logger zerolog.Logger
	done   chan struct{}
}

// NewRejectionConsumer creates new Kafka consumer for rejection events
func NewRejectionConsumer(cfg *config.KafkaConfig, sender deps.TelegramSender, logger zerolog.Logger) *RejectionConsumer {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	topics := []string{
		cfg.TopicSubscriptionRejected,
		cfg.TopicUnsubscriptionRejected,
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     cfg.GroupID + "-rejection",
		GroupTopics: topics,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
	})

	logger.Info().
		Strs("brokers", brokers).
		Str("group_id", cfg.GroupID+"-rejection").
		Strs("topics", topics).
		Msg("Kafka rejection consumer initialized")

	return &RejectionConsumer{
		reader: reader,
		sender: sender,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start starts consuming rejection messages from Kafka
func (c *RejectionConsumer) Start(ctx context.Context) {
	c.logger.Info().Msg("Starting Kafka rejection consumer...")

	go func() {
		for {
			select {
			case <-c.done:
				c.logger.Info().Msg("Kafka rejection consumer stopped by done signal")
				return
			case <-ctx.Done():
				c.logger.Info().Msg("Kafka rejection consumer stopped by context cancellation")
				return
			default:
				msg, err := c.reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					c.logger.Error().Err(err).Msg("Failed to read rejection message from Kafka")
					continue
				}

				c.logger.Debug().
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Received rejection message from Kafka")

				if err := c.handleRejection(ctx, msg.Value); err != nil {
					c.logger.Error().Err(err).Msg("Failed to handle rejection event")
				}
			}
		}
	}()
}

// handleRejection processes rejection event and sends notification to user
func (c *RejectionConsumer) handleRejection(ctx context.Context, data []byte) error {
	var event dto.RejectedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal rejection event: %w", err)
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse user_id: %w", err)
	}

	var message string
	switch event.Type {
	case "subscription_rejected":
		channelDisplay := event.ChannelName
		if channelDisplay == "" {
			channelDisplay = event.ChannelID
		}
		message = fmt.Sprintf("❌ Не удалось подписаться на канал %s", channelDisplay)
		if event.Reason != "" {
			message += fmt.Sprintf(": %s", event.Reason)
		}

	case "unsubscription_rejected":
		channelDisplay := event.ChannelName
		if channelDisplay == "" {
			channelDisplay = event.ChannelID
		}
		message = fmt.Sprintf("❌ Не удалось отписаться от канала %s", channelDisplay)
		if event.Reason != "" {
			message += fmt.Sprintf(": %s", event.Reason)
		}

	default:
		c.logger.Warn().Str("type", event.Type).Msg("Unknown rejection event type")
		return nil
	}

	c.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", event.ChannelID).
		Str("type", event.Type).
		Msg("Sending rejection notification to user")

	return c.sender.SendMessage(ctx, userID, message)
}

// Stop stops the consumer gracefully
func (c *RejectionConsumer) Stop() error {
	c.logger.Info().Msg("Stopping Kafka rejection consumer...")
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka rejection reader")
		return err
	}

	c.logger.Info().Msg("Kafka rejection consumer stopped successfully")
	return nil
}
