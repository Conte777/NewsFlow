package kafka

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/news-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/delivery/kafka"
)

const (
	minBytes = 1     // 1 byte - читать сообщения сразу
	maxBytes = 10e6  // 10MB
)

type Consumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
}

func NewConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.TopicNewsReceived,
		GroupID:  cfg.GroupID,
		MinBytes: minBytes,
		MaxBytes: maxBytes,
		MaxWait:  3 * time.Second,
	})

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.TopicNewsReceived).
		Str("group_id", cfg.GroupID).
		Msg("Kafka consumer initialized")

	return &Consumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	go c.consume(ctx)
	c.logger.Info().Msg("Kafka consumer started")
}

func (c *Consumer) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info().Msg("Consumer context canceled, stopping")
			return
		case <-c.done:
			c.logger.Info().Msg("Consumer received stop signal")
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error().Err(err).Msg("Failed to fetch message")
				continue
			}

			c.logger.Debug().
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Received message from Kafka")

			if err := c.handlers.HandleNewsReceived(ctx, msg.Value); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to handle news received event")
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to commit message")
			}
		}
	}
}

func (c *Consumer) Stop() error {
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka consumer")
		return err
	}

	c.logger.Info().Msg("Kafka consumer stopped")
	return nil
}
