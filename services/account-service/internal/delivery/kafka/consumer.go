package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

const (
	// Consumer group ID for account service
	consumerGroupID = "account-service-group"

	// Topics for subscription events
	topicSubscriptionCreated = "subscriptions.created"
	topicSubscriptionDeleted = "subscriptions.deleted"
)

// KafkaConsumer consumes subscription events from Kafka
type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string
	logger        zerolog.Logger
	handler       domain.SubscriptionEventHandler
	closeOnce     sync.Once
	closeErr      error
}

// ConsumerConfig holds configuration for Kafka consumer
type ConsumerConfig struct {
	Brokers       []string                        // Kafka broker addresses
	Logger        zerolog.Logger                  // Logger for monitoring
	Handler       domain.SubscriptionEventHandler // Event handler
	SessionTimeout time.Duration                  // Session timeout (default: 10s)
	HeartbeatInterval time.Duration               // Heartbeat interval (default: 3s)
}

// NewKafkaConsumer creates a new Kafka consumer for subscription events
//
// Configuration as per ACC-2.4:
// - Consumer group: account-service-group
// - Topics: subscriptions.created, subscriptions.deleted
// - Session timeout: 10s
// - Heartbeat interval: 3s
// - Max poll records: 100
// - Auto commit: disabled (manual commit for at-least-once processing)
func NewKafkaConsumer(cfg ConsumerConfig) (domain.KafkaConsumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("no kafka brokers specified")
	}
	if cfg.Handler == nil {
		return nil, fmt.Errorf("event handler is required")
	}

	// Set defaults
	if cfg.SessionTimeout == 0 {
		cfg.SessionTimeout = 10 * time.Second
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 3 * time.Second
	}

	config := sarama.NewConfig()

	// Consumer group configuration
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	// Session configuration (ACC-2.4)
	config.Consumer.Group.Session.Timeout = cfg.SessionTimeout
	config.Consumer.Group.Heartbeat.Interval = cfg.HeartbeatInterval

	// Max poll records configuration (ACC-2.4)
	config.Consumer.MaxProcessingTime = 30 * time.Second
	config.ChannelBufferSize = 100 // Max poll records

	// Manual commit for at-least-once processing (ACC-2.4: auto commit disabled)
	config.Consumer.Offsets.AutoCommit.Enable = false

	// Return errors for proper handling
	config.Consumer.Return.Errors = true

	// Kafka version compatibility
	config.Version = sarama.V2_6_0_0

	// Client ID for identification
	config.ClientID = "account-service-consumer"

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, consumerGroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	topics := []string{topicSubscriptionCreated, topicSubscriptionDeleted}

	kc := &KafkaConsumer{
		consumerGroup: consumerGroup,
		topics:        topics,
		logger:        cfg.Logger,
		handler:       cfg.Handler,
	}

	cfg.Logger.Info().
		Strs("brokers", cfg.Brokers).
		Strs("topics", topics).
		Str("group_id", consumerGroupID).
		Dur("session_timeout", cfg.SessionTimeout).
		Dur("heartbeat_interval", cfg.HeartbeatInterval).
		Msg("Kafka consumer initialized successfully")

	return kc, nil
}

// ConsumeSubscriptionEvents starts consuming subscription events from Kafka
//
// This method blocks until the context is cancelled or an error occurs.
// It processes messages from subscriptions.created and subscriptions.deleted topics.
func (kc *KafkaConsumer) ConsumeSubscriptionEvents(ctx context.Context, handler domain.SubscriptionEventHandler) error {
	if handler != nil {
		kc.handler = handler
	}

	if kc.handler == nil {
		return fmt.Errorf("event handler is required")
	}

	// Create consumer group handler
	cgHandler := &consumerGroupHandler{
		logger:  kc.logger,
		handler: kc.handler,
	}

	// Error handling goroutine
	errChan := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-kc.consumerGroup.Errors():
				if !ok {
					// Channel closed
					return
				}
				kc.logger.Error().Err(err).Msg("Consumer group error")
				select {
				case errChan <- err:
				default:
				}
			}
		}
	}()

	// Consume messages in a loop (handles rebalancing)
	kc.logger.Info().Msg("Starting to consume subscription events")

	for {
		select {
		case <-ctx.Done():
			kc.logger.Info().Msg("Context cancelled, stopping consumer")
			return ctx.Err()
		case err := <-errChan:
			kc.logger.Error().Err(err).Msg("Received error from consumer group")
			return err
		default:
			// Consume will start a consumer group session and blocks until:
			// - Session is finished (rebalancing)
			// - Context is cancelled
			// - Error occurs
			if err := kc.consumerGroup.Consume(ctx, kc.topics, cgHandler); err != nil {
				kc.logger.Error().Err(err).Msg("Error from consumer group Consume")
				return fmt.Errorf("consume failed: %w", err)
			}

			// Check if context was cancelled
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
}

// Close gracefully shuts down the Kafka consumer
func (kc *KafkaConsumer) Close() error {
	kc.closeOnce.Do(func() {
		kc.logger.Info().Msg("Closing Kafka consumer")

		if err := kc.consumerGroup.Close(); err != nil {
			kc.logger.Error().Err(err).Msg("Error closing consumer group")
			kc.closeErr = fmt.Errorf("failed to close consumer group: %w", err)
			return
		}

		kc.logger.Info().Msg("Kafka consumer closed successfully")
	})

	return kc.closeErr
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	logger  zerolog.Logger
	handler domain.SubscriptionEventHandler
}

// Setup is called when a new session is established
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.logger.Info().
		Int32("generation_id", session.GenerationID()).
		Str("member_id", session.MemberID()).
		Msg("Consumer group session started")
	return nil
}

// Cleanup is called when a session is finished
func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.logger.Info().
		Int32("generation_id", session.GenerationID()).
		Msg("Consumer group session ended")
	return nil
}

// ConsumeClaim processes messages from a partition
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := session.Context()

	h.logger.Debug().
		Str("topic", claim.Topic()).
		Int32("partition", claim.Partition()).
		Int64("initial_offset", claim.InitialOffset()).
		Msg("Started consuming partition")

	for {
		select {
		case <-ctx.Done():
			h.logger.Debug().Msg("Session context cancelled")
			return nil

		case msg, ok := <-claim.Messages():
			if !ok {
				h.logger.Debug().Msg("Message channel closed")
				return nil
			}

			// Process message
			if err := h.processMessage(ctx, msg); err != nil {
				h.logger.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int32("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Failed to process message - SKIPPING (message will be committed)")

				// Commit offset even on error to avoid getting stuck on bad messages
				// This is "best-effort" approach - failed messages are logged but skipped
				// For production, consider sending to DLQ (Dead Letter Queue)
				session.MarkMessage(msg, "")
				continue
			}

			// Commit offset manually (auto-commit is disabled)
			session.MarkMessage(msg, "")

			h.logger.Debug().
				Str("topic", msg.Topic).
				Int32("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("Message processed and marked")
		}
	}
}

// processMessage processes a single Kafka message
func (h *consumerGroupHandler) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// Parse subscription event
	var event domain.SubscriptionEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal subscription event: %w", err)
	}

	h.logger.Debug().
		Str("event_type", event.EventType).
		Int64("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("Processing subscription event")

	// Route to appropriate handler based on topic
	switch msg.Topic {
	case topicSubscriptionCreated:
		return h.handler.HandleSubscriptionCreated(ctx, event.UserID, event.ChannelID, event.ChannelName)

	case topicSubscriptionDeleted:
		return h.handler.HandleSubscriptionDeleted(ctx, event.UserID, event.ChannelID)

	default:
		return fmt.Errorf("unknown topic: %s", msg.Topic)
	}
}
