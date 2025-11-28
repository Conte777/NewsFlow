package kafka

import (
	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	logger   zerolog.Logger
}

func NewKafkaProducer(brokers []string, logger zerolog.Logger) (*KafkaProducer, error) {

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 5
	config.Producer.Retry.Backoff = 100
	config.Producer.RequiredAcks = sarama.WaitForAll

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create Kafka SyncProducer")
		return nil, err
	}

	logger.Info().Msg("Kafka SyncProducer successfully initialized")

	return &KafkaProducer{
		producer: producer,
		logger:   logger,
	}, nil
}

func (p *KafkaProducer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}
