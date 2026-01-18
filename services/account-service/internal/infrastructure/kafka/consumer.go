package kafka

import (
	"context"
	"encoding/json"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
)

const maxRetries = 3

// KafkaConsumer handles consumption of subscription events from Kafka
type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string
	sagaHandler   deps.SagaEventHandler
	logger        zerolog.Logger
}

// NewKafkaConsumer creates a new Kafka consumer for subscription events
func NewKafkaConsumer(
	brokers []string,
	groupID string,
	topics []string,
	sagaHandler deps.SagaEventHandler,
	logger zerolog.Logger,
) (*KafkaConsumer, error) {
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
		sagaHandler:   sagaHandler,
		logger:        logger,
	}, nil
}

// Start begins consuming messages in a goroutine
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

// Close gracefully shuts down the consumer
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

// Setup is called at the beginning of a new session
func (c *KafkaConsumer) Setup(session sarama.ConsumerGroupSession) error {
	c.logger.Info().
		Str("member_id", session.MemberID()).
		Msg("consumer group session setup completed")
	return nil
}

// Cleanup is called at the end of a session
func (c *KafkaConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
	c.logger.Info().
		Str("member_id", session.MemberID()).
		Msg("consumer group session cleanup completed")
	return nil
}

// ConsumeClaim processes messages from a partition
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

		if err := c.processMessage(session.Context(), msg); err != nil {
			c.logger.Error().
				Err(err).
				Str("topic", msg.Topic).
				Int64("offset", msg.Offset).
				Msg("all retry attempts failed, skipping message")
			continue
		}

		session.MarkMessage(msg, "")
	}

	c.logger.Info().
		Str("topic", claim.Topic()).
		Int32("partition", claim.Partition()).
		Msg("stopped message consumption from partition")

	return nil
}

func (c *KafkaConsumer) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = c.handleMessage(ctx, msg)
		if lastErr == nil {
			return nil
		}

		c.logger.Warn().
			Err(lastErr).
			Int("attempt", attempt).
			Int("max_attempts", maxRetries).
			Str("topic", msg.Topic).
			Msg("handler failed to process event, retrying")
	}

	return lastErr
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	switch msg.Topic {
	// Saga: Subscription flow
	case "subscription.pending":
		return c.handleSubscriptionPending(ctx, msg)

	// Saga: Unsubscription flow
	case "unsubscription.pending":
		return c.handleUnsubscriptionPending(ctx, msg)

	default:
		c.logger.Warn().
			Str("topic", msg.Topic).
			Msg("received message from unknown topic")
		return nil
	}
}

func (c *KafkaConsumer) handleSubscriptionPending(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event SubscriptionPendingEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error().
			Err(err).
			Str("topic", msg.Topic).
			Msg("failed to unmarshal subscription.pending event")
		return nil // Don't retry unmarshal errors
	}

	userID, err := event.GetUserIDInt64()
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid user_id in subscription.pending event")
		return nil
	}

	c.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", event.ChannelID).
		Str("channel_name", event.ChannelName).
		Msg("processing subscription.pending event")

	return c.sagaHandler.HandleSubscriptionPending(ctx, userID, event.ChannelID, event.ChannelName)
}

func (c *KafkaConsumer) handleUnsubscriptionPending(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event UnsubscriptionPendingEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error().
			Err(err).
			Str("topic", msg.Topic).
			Msg("failed to unmarshal unsubscription.pending event")
		return nil // Don't retry unmarshal errors
	}

	userID, err := event.GetUserIDInt64()
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("user_id", event.UserID).
			Msg("invalid user_id in unsubscription.pending event")
		return nil
	}

	c.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", event.ChannelID).
		Msg("processing unsubscription.pending event")

	return c.sagaHandler.HandleUnsubscriptionPending(ctx, userID, event.ChannelID)
}
