// Package workers contains background workers for the bot domain
package workers

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/delivery/kafka"
)

// NewsConsumer consumes news delivery events from Kafka
type NewsConsumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewNewsConsumer creates new Kafka consumer for news delivery
func NewNewsConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *NewsConsumer {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  cfg.GroupID,
		Topic:    "news.deliver",
		MinBytes: 1,    // 1 byte - return immediately when message available
		MaxBytes: 10e6, // 10MB
	})

	logger.Info().
		Strs("brokers", brokers).
		Str("group_id", cfg.GroupID).
		Str("topic", "news.deliver").
		Msg("Kafka news consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &NewsConsumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts consuming messages from Kafka
func (c *NewsConsumer) Start() {
	c.logger.Info().Msg("Starting Kafka news consumer...")

	go func() {
		for {
			select {
			case <-c.done:
				c.logger.Info().Msg("Kafka news consumer stopped by done signal")
				return
			case <-c.ctx.Done():
				c.logger.Info().Msg("Kafka news consumer stopped by context cancellation")
				return
			default:
				msg, err := c.reader.ReadMessage(c.ctx)
				if err != nil {
					if c.ctx.Err() != nil {
						return
					}
					c.logger.Error().Err(err).Msg("Failed to read message from Kafka")
					continue
				}

				c.logger.Debug().
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Received message from Kafka")

				if err := c.handlers.HandleNewsDelivery(c.ctx, msg.Value); err != nil {
					c.logger.Error().Err(err).Msg("Failed to handle news delivery")
				}
			}
		}
	}()
}

// Stop stops the consumer gracefully
func (c *NewsConsumer) Stop() error {
	c.logger.Info().Msg("Stopping Kafka news consumer...")
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka reader")
		return err
	}

	c.logger.Info().Msg("Kafka news consumer stopped successfully")
	return nil
}
