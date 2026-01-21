package kafka

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/news-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/delivery/kafka"
)

// NewsDeliveredConsumer consumes news.delivered events from bot-service
type NewsDeliveredConsumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewNewsDeliveredConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *NewsDeliveredConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.TopicNewsDelivered,
		GroupID:     cfg.GroupIDDelivered,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.TopicNewsDelivered).
		Str("group_id", cfg.GroupIDDelivered).
		Msg("News delivered consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &NewsDeliveredConsumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *NewsDeliveredConsumer) Start() {
	go c.consume()
	c.logger.Info().Msg("News delivered consumer started")
}

func (c *NewsDeliveredConsumer) consume() {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info().Msg("News delivered consumer context canceled, stopping")
			return
		case <-c.done:
			c.logger.Info().Msg("News delivered consumer received stop signal")
			return
		default:
			msg, err := c.reader.FetchMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				c.logger.Error().Err(err).Msg("Failed to fetch delivered message")
				continue
			}

			c.logger.Debug().
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Received delivered message from Kafka")

			if err := c.handlers.HandleDeliveryConfirmation(c.ctx, msg.Value); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to handle delivery confirmation event")
				continue
			}

			if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to commit delivered message")
			}
		}
	}
}

func (c *NewsDeliveredConsumer) Stop() error {
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close news delivered consumer")
		return err
	}

	c.logger.Info().Msg("News delivered consumer stopped")
	return nil
}
