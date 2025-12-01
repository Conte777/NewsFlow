package kafka

import (
	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	logger        zerolog.Logger
	handler       sarama.ConsumerGroupHandler
	topics        []string
	groupID       string
}

func NewKafkaConsumer(
	brokers []string,
	groupID string,
	topics []string,
	handler sarama.ConsumerGroupHandler,
	logger zerolog.Logger,
) (*KafkaConsumer, error) {

	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		logger.Error().
			Err(err).
			Str("group_id", groupID).
			Msg("failed to create Kafka consumer group")
		return nil, err
	}

	logger.Info().
		Str("group_id", groupID).
		Strs("topics", topics).
		Msg("Kafka consumer group successfully initialized")

	return &KafkaConsumer{
		consumerGroup: consumerGroup,
		logger:        logger,
		handler:       handler,
		topics:        topics,
		groupID:       groupID,
	}, nil
}
