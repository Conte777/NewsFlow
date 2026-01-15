package kafka

import (
	"context"
	"encoding/json"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/dto"
	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

type ConsumerEventHandler interface {
	Handle(event *dto.SubscriptionEvent) error
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

func (c *KafkaConsumer) Close() error {
	if c.consumerGroup == nil {
		c.logger.Info().Msg("Kafka consumer group is already closed or not initialized")
		return nil
	}

	c.logger.Info().Msg("closing Kafka consumer group...")

	if err := c.consumerGroup.Close(); err != nil {
		c.logger.Error().Err(err).Msg("failed to close Kafka consumer group")
		return err
	}

	c.logger.Info().Msg("Kafka consumer group successfully closed")
	return nil
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

const maxRetries = 3

func (c *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	c.logger.Info().
		Str("topic", claim.Topic()).
		Int32("partition", claim.Partition()).
		Msg("starting message consumption from partition")

	for msg := range claim.Messages() {
		c.logger.Debug().
			Str("topic", msg.Topic).
			Int32("partition", msg.Partition).
			Int64("offset", msg.Offset).
			Msg("received message from Kafka")

		var event dto.SubscriptionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Str("key", string(msg.Key)).
				Msg("failed to unmarshal subscription event")
			continue
		}

		var err error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = c.handler.Handle(&event)
			if err == nil {
				break
			}

			c.logger.Error().
				Err(err).
				Int("attempt", attempt).
				Int("max_attempts", maxRetries).
				Str("topic", msg.Topic).
				Str("user_id", event.UserID).
				Msg("handler failed to process event, retrying")
		}

		if err != nil {
			c.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Str("user_id", event.UserID).
				Msg("all retry attempts failed, message will be reprocessed later")
			continue
		}

		session.MarkMessage(msg, "")
		c.logger.Info().
			Str("topic", msg.Topic).
			Str("user_id", event.UserID).
			Int64("offset", msg.Offset).
			Msg("subscription event successfully processed")
	}

	c.logger.Info().
		Str("topic", claim.Topic()).
		Int32("partition", claim.Partition()).
		Msg("stopped message consumption from partition")

	return nil
}

func (c *KafkaConsumer) Start(ctx context.Context) {
	go func() {
		for {
			if ctx.Err() != nil {
				c.logger.Info().Msg("consumer context canceled, stopping consumer group")
				return
			}

			if err := c.consumerGroup.Consume(ctx, c.topics, c); err != nil {
				c.logger.Error().Err(err).Msg("error from consumer group")
			}
		}
	}()

	c.logger.Info().
		Strs("topics", c.topics).
		Msg("Kafka consumer group started")
}
