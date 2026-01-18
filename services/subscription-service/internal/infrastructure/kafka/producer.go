package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

type KafkaProducer struct {
	producer     sarama.SyncProducer
	logger       zerolog.Logger
	successCount uint64
	errorCount   uint64
}

func NewKafkaProducer(brokers []string, logger zerolog.Logger) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 5
	config.Producer.Retry.Backoff = 500 * time.Millisecond
	config.Producer.Timeout = 10 * time.Second
	config.Producer.RequiredAcks = sarama.WaitForAll

	sarama.Logger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create Kafka SyncProducer")
		return nil, err
	}

	logger.Info().Msg("Kafka SyncProducer successfully initialized with retry/backoff config")

	return &KafkaProducer{
		producer: producer,
		logger:   logger,
	}, nil
}

func (p *KafkaProducer) Close() error {
	if p.producer == nil {
		p.logger.Info().Msg("Kafka producer already closed or not initialized")
		return nil
	}

	err := p.producer.Close()
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to close Kafka producer")
		return err
	}

	p.logger.Info().Msg("Kafka producer successfully closed")
	return nil
}

// SendToTopic sends any event to a specific topic
func (p *KafkaProducer) SendToTopic(ctx context.Context, topic string, key string, event any) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		p.errorCount++
		p.logger.Error().
			Err(err).
			Str("topic", topic).
			Uint64("error_count", p.errorCount).
			Msg("failed to marshal event")
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(bytes),
	}

	start := time.Now()
	partition, offset, err := p.producer.SendMessage(msg)
	latency := time.Since(start)

	if err != nil {
		p.errorCount++
		p.logger.Error().
			Err(err).
			Str("topic", topic).
			Str("key", key).
			Dur("latency", latency).
			Uint64("error_count", p.errorCount).
			Msg("failed to send event to kafka")
		return err
	}

	p.successCount++

	p.logger.Info().
		Str("topic", topic).
		Str("key", key).
		Int32("partition", partition).
		Int64("offset", offset).
		Dur("latency", latency).
		Uint64("success_count", p.successCount).
		Msg("event sent to kafka")

	return nil
}
