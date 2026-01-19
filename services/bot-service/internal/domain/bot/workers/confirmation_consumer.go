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
	subReader   *kafka.Reader
	unsubReader *kafka.Reader
	sender      deps.TelegramSender
	logger      zerolog.Logger
	done        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewConfirmationConsumer creates new Kafka consumer for confirmation events
func NewConfirmationConsumer(cfg *config.KafkaConfig, sender deps.TelegramSender, logger zerolog.Logger) *ConfirmationConsumer {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	subReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     cfg.GroupID + "-sub-confirmation",
		Topic:       cfg.TopicSubscriptionConfirmed,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})

	unsubReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     cfg.GroupID + "-unsub-confirmation",
		Topic:       cfg.TopicUnsubscriptionConfirmed,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info().
		Strs("brokers", brokers).
		Str("sub_topic", cfg.TopicSubscriptionConfirmed).
		Str("unsub_topic", cfg.TopicUnsubscriptionConfirmed).
		Msg("Kafka confirmation consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &ConfirmationConsumer{
		subReader:   subReader,
		unsubReader: unsubReader,
		sender:      sender,
		logger:      logger,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts consuming confirmation messages from Kafka
func (c *ConfirmationConsumer) Start() {
	c.logger.Info().Msg("Starting Kafka confirmation consumer...")

	// Start subscription confirmation reader
	go c.consumeFromReader(c.subReader, "subscription")

	// Start unsubscription confirmation reader
	go c.consumeFromReader(c.unsubReader, "unsubscription")
}

func (c *ConfirmationConsumer) consumeFromReader(reader *kafka.Reader, readerName string) {
	for {
		select {
		case <-c.done:
			c.logger.Info().Str("reader", readerName).Msg("Kafka confirmation consumer stopped by done signal")
			return
		case <-c.ctx.Done():
			c.logger.Info().Str("reader", readerName).Msg("Kafka confirmation consumer stopped by context cancellation")
			return
		default:
			msg, err := reader.FetchMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				c.logger.Error().Err(err).Str("reader", readerName).Msg("Failed to fetch confirmation message from Kafka")
				continue
			}

			c.logger.Debug().
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Received confirmation message from Kafka")

			if err := c.handleConfirmation(c.ctx, msg.Value); err != nil {
				c.logger.Error().Err(err).Msg("Failed to handle confirmation event")
			}

			if err := reader.CommitMessages(c.ctx, msg); err != nil {
				c.logger.Error().Err(err).Msg("Failed to commit confirmation message")
			}
		}
	}
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

	if err := c.sender.SendChatAction(ctx, userID, "typing"); err != nil {
		c.logger.Warn().Int64("user_id", userID).Err(err).Msg("Failed to send typing indicator")
	}

	return c.sender.SendMessage(ctx, userID, message)
}

// Stop stops the consumer gracefully
func (c *ConfirmationConsumer) Stop() error {
	c.logger.Info().Msg("Stopping Kafka confirmation consumer...")
	c.cancel()
	close(c.done)

	if err := c.subReader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka subscription confirmation reader")
		return err
	}

	if err := c.unsubReader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka unsubscription confirmation reader")
		return err
	}

	c.logger.Info().Msg("Kafka confirmation consumer stopped successfully")
	return nil
}
