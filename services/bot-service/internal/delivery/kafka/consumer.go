package kafka

import (
	"context"
	"fmt"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	logger zerolog.Logger
	config config.KafkaConfig
}

// NewConsumer создает нового Kafka consumer
func NewConsumer(cfg config.KafkaConfig, logger zerolog.Logger) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("no brokers configured")
	}

	// Топик должен передаваться отдельно или быть в конфиге
	// Если в вашем конфиге нет Topic, передавайте его как параметр
	readerConfig := kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		// GroupID и Topic должны быть установлены вызывающей стороной
		// или взяты из конфига если они там есть
	}

	reader := kafka.NewReader(readerConfig)

	return &Consumer{
		reader: reader,
		logger: logger,
		config: cfg,
	}, nil
}

// ReadMessage читает сообщение из Kafka
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to read message from Kafka")
		return kafka.Message{}, err
	}

	c.logger.Debug().
		Str("topic", msg.Topic).
		Str("key", string(msg.Key)).
		Int("value_len", len(msg.Value)).
		Msg("Message received")

	return msg, nil
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
