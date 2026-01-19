// Package workers contains background workers for the bot domain
package workers

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/delivery/kafka"
)

// DeleteConsumer consumes news delete events from Kafka
type DeleteConsumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewDeleteConsumer creates new Kafka consumer for news delete events
func NewDeleteConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *DeleteConsumer {
	brokers := cfg.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9093"}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  cfg.GroupID + "-delete",
		Topic:    "news.delete",
		MinBytes: 1,    // 1 byte - return immediately when message available
		MaxBytes: 10e6, // 10MB
	})

	logger.Info().
		Strs("brokers", brokers).
		Str("group_id", cfg.GroupID+"-delete").
		Str("topic", "news.delete").
		Msg("Kafka delete consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &DeleteConsumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts consuming messages from Kafka
func (c *DeleteConsumer) Start() {
	c.logger.Info().Msg("Starting Kafka delete consumer...")

	go func() {
		for {
			select {
			case <-c.done:
				c.logger.Info().Msg("Kafka delete consumer stopped by done signal")
				return
			case <-c.ctx.Done():
				c.logger.Info().Msg("Kafka delete consumer stopped by context cancellation")
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
					Msg("Received delete message from Kafka")

				if err := c.handlers.HandleNewsDelete(c.ctx, msg.Value); err != nil {
					c.logger.Error().Err(err).Msg("Failed to handle news delete")
				}
			}
		}
	}()
}

// Stop stops the consumer gracefully
func (c *DeleteConsumer) Stop() error {
	c.logger.Info().Msg("Stopping Kafka delete consumer...")
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka reader")
		return err
	}

	c.logger.Info().Msg("Kafka delete consumer stopped successfully")
	return nil
}
