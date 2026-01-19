// Package workers contains background workers for the bot domain
package workers

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/delivery/kafka"
)

// EditConsumer consumes news edit events from Kafka
type EditConsumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewEditConsumer creates new Kafka consumer for news edit events
func NewEditConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *EditConsumer {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  cfg.GroupID + "-edit",
		Topic:    "news.edit",
		MinBytes: 1,    // 1 byte - return immediately when message available
		MaxBytes: 10e6, // 10MB
	})

	logger.Info().
		Strs("brokers", brokers).
		Str("group_id", cfg.GroupID+"-edit").
		Str("topic", "news.edit").
		Msg("Kafka edit consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &EditConsumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts consuming messages from Kafka
func (c *EditConsumer) Start() {
	c.logger.Info().Msg("Starting Kafka edit consumer...")

	go func() {
		for {
			select {
			case <-c.done:
				c.logger.Info().Msg("Kafka edit consumer stopped by done signal")
				return
			case <-c.ctx.Done():
				c.logger.Info().Msg("Kafka edit consumer stopped by context cancellation")
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
					Msg("Received edit message from Kafka")

				if err := c.handlers.HandleNewsEdit(c.ctx, msg.Value); err != nil {
					c.logger.Error().Err(err).Msg("Failed to handle news edit")
				}
			}
		}
	}()
}

// Stop stops the consumer gracefully
func (c *EditConsumer) Stop() error {
	c.logger.Info().Msg("Stopping Kafka edit consumer...")
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka reader")
		return err
	}

	c.logger.Info().Msg("Kafka edit consumer stopped successfully")
	return nil
}
