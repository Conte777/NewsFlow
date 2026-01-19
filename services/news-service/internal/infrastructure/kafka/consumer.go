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
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.TopicNewsReceived,
		GroupID:     cfg.GroupID,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		MaxWait:     3 * time.Second,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.TopicNewsReceived).
		Str("group_id", cfg.GroupID).
		Msg("Kafka consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *Consumer) Start() {
	go c.consume()
	c.logger.Info().Msg("Kafka consumer started")
}

func (c *Consumer) consume() {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info().Msg("Consumer context canceled, stopping")
			return
		case <-c.done:
			c.logger.Info().Msg("Consumer received stop signal")
			return
		default:
			msg, err := c.reader.FetchMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
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

			if err := c.handlers.HandleNewsReceived(c.ctx, msg.Value); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to handle news received event")
				continue
			}

			if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to commit message")
			}
		}
	}
}

func (c *Consumer) Stop() error {
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close Kafka consumer")
		return err
	}

	c.logger.Info().Msg("Kafka consumer stopped")
	return nil
}
