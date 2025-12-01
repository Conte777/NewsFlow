package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

type ConsumerEventHandler interface {
	HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error
}

type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string
	logger        zerolog.Logger
	handler       ConsumerEventHandler
}

func NewKafkaConsumer(
	brokers []string,
	groupID string,
	topics []string,
	handler ConsumerEventHandler,
	logger zerolog.Logger) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create Kafka consumer group")
		return nil, err
	}

	logger.Info().
		Str("group_id", groupID).
		Strs("topics", topics).
		Msg("Kafka consumer group successfully initialized")

	return &KafkaConsumer{
		consumerGroup: consumerGroup,
		topics:        topics,
		logger:        logger,
		handler:       handler,
	}, nil
}

func (c *KafkaConsumer) Setup(session sarama.ConsumerGroupSession) error {
	c.logger.Info().
		Str("member_id", session.MemberID()).
		Msg("consumer group session setup completed")
	return nil
}

func (c *KafkaConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
	c.logger.Info().
		Str("member_id", session.MemberID()).
		Msg("consumer group session cleanup completed")
	return nil
}

func (c *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	c.logger.Info().
		Int32("partition", claim.Partition()).
		Msg("starting message consumption for partition")

	for msg := range claim.Messages() {
		err := c.handler.HandleMessage(session.Context(), msg)
		if err != nil {
			c.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Int32("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("failed to process message")
			continue
		}

		session.MarkMessage(msg, "")
	}

	return nil
}
