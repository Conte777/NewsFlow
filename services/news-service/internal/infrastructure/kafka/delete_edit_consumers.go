package kafka

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/Conte777/NewsFlow/services/news-service/config"
	kafkaHandlers "github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/delivery/kafka"
)

// NewsDeletedConsumer consumes news.deleted events from account-service
type NewsDeletedConsumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewNewsDeletedConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *NewsDeletedConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       "news.deleted",
		GroupID:     cfg.GroupID,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", "news.deleted").
		Str("group_id", cfg.GroupID).
		Msg("News deleted consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &NewsDeletedConsumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *NewsDeletedConsumer) Start() {
	go c.consume()
	c.logger.Info().Msg("News deleted consumer started")
}

func (c *NewsDeletedConsumer) consume() {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info().Msg("News deleted consumer context canceled, stopping")
			return
		case <-c.done:
			c.logger.Info().Msg("News deleted consumer received stop signal")
			return
		default:
			msg, err := c.reader.FetchMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				c.logger.Error().Err(err).Msg("Failed to fetch deleted message")
				continue
			}

			c.logger.Debug().
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Received deleted message from Kafka")

			if err := c.handlers.HandleNewsDeleted(c.ctx, msg.Value); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to handle news deleted event")
				continue
			}

			if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to commit deleted message")
			}
		}
	}
}

func (c *NewsDeletedConsumer) Stop() error {
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close news deleted consumer")
		return err
	}

	c.logger.Info().Msg("News deleted consumer stopped")
	return nil
}

// NewsEditedConsumer consumes news.edited events from account-service
type NewsEditedConsumer struct {
	reader   *kafka.Reader
	handlers *kafkaHandlers.Handlers
	logger   zerolog.Logger
	done     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewNewsEditedConsumer(cfg *config.KafkaConfig, handlers *kafkaHandlers.Handlers, logger zerolog.Logger) *NewsEditedConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       "news.edited",
		GroupID:     cfg.GroupID,
		MinBytes:    minBytes,
		MaxBytes:    maxBytes,
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset,
	})

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", "news.edited").
		Str("group_id", cfg.GroupID).
		Msg("News edited consumer initialized")

	ctx, cancel := context.WithCancel(context.Background())

	return &NewsEditedConsumer{
		reader:   reader,
		handlers: handlers,
		logger:   logger,
		done:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *NewsEditedConsumer) Start() {
	go c.consume()
	c.logger.Info().Msg("News edited consumer started")
}

func (c *NewsEditedConsumer) consume() {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info().Msg("News edited consumer context canceled, stopping")
			return
		case <-c.done:
			c.logger.Info().Msg("News edited consumer received stop signal")
			return
		default:
			msg, err := c.reader.FetchMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				c.logger.Error().Err(err).Msg("Failed to fetch edited message")
				continue
			}

			c.logger.Debug().
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Received edited message from Kafka")

			if err := c.handlers.HandleNewsEdited(c.ctx, msg.Value); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to handle news edited event")
				continue
			}

			if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
				c.logger.Error().Err(err).
					Int64("offset", msg.Offset).
					Msg("Failed to commit edited message")
			}
		}
	}
}

func (c *NewsEditedConsumer) Stop() error {
	c.cancel()
	close(c.done)

	if err := c.reader.Close(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to close news edited consumer")
		return err
	}

	c.logger.Info().Msg("News edited consumer stopped")
	return nil
}
