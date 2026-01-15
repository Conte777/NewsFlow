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
	handler       deps.SubscriptionEventHandler
	logger        zerolog.Logger
}

// NewKafkaConsumer creates a new Kafka consumer for subscription events
func NewKafkaConsumer(
	brokers []string,
	groupID string,
	topics []string,
	handler deps.SubscriptionEventHandler,
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
		handler:       handler,
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

// processMessage handles a single message with retry logic
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

// handleMessage routes the message to the appropriate handler based on topic
func (c *KafkaConsumer) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	switch msg.Topic {
	case "subscriptions.created":
		return c.handleSubscriptionCreated(ctx, msg)
	case "subscriptions.deleted":
		return c.handleSubscriptionDeleted(ctx, msg)
	default:
		c.logger.Warn().
			Str("topic", msg.Topic).
			Msg("received message from unknown topic")
		return nil
	}
}

// handleSubscriptionCreated processes subscription created events
func (c *KafkaConsumer) handleSubscriptionCreated(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event SubscriptionCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error().
			Err(err).
			Str("topic", msg.Topic).
			Msg("failed to unmarshal subscription created event")
		return nil // Don't retry unmarshal errors
	}

	c.logger.Info().
		Int64("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Str("channel_name", event.ChannelName).
		Msg("processing subscription created event")

	return c.handler.HandleSubscriptionCreated(ctx, event.UserID, event.ChannelID, event.ChannelName)
}

// handleSubscriptionDeleted processes subscription deleted events
func (c *KafkaConsumer) handleSubscriptionDeleted(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event SubscriptionDeletedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error().
			Err(err).
			Str("topic", msg.Topic).
			Msg("failed to unmarshal subscription deleted event")
		return nil // Don't retry unmarshal errors
	}

	c.logger.Info().
		Int64("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("processing subscription deleted event")

	return c.handler.HandleSubscriptionDeleted(ctx, event.UserID, event.ChannelID)
}
