package kafka

import (
	"context"
	"encoding/json"
	"errors"
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

var (
	// ErrInvalidMessage indicates message cannot be processed (permanent error)
	ErrInvalidMessage = errors.New("invalid message format")

	// ErrUnknownTopic indicates unknown topic (permanent error)
	ErrUnknownTopic = errors.New("unknown topic")
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
	// Use OffsetOldest to ensure all messages are processed from the beginning
	// when consumer group starts for the first time
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

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
//
// The handler is set during consumer creation and cannot be changed during runtime
// to avoid race conditions.
func (kc *KafkaConsumer) ConsumeSubscriptionEvents(ctx context.Context, handler domain.SubscriptionEventHandler) error {
	// Note: handler parameter is kept for interface compatibility but ignored
	// to prevent race conditions. Use handler from constructor instead.
	if handler != nil && kc.handler == nil {
		return fmt.Errorf("handler should be set in constructor, not at runtime")
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
				// Check if error is retryable or permanent
				if isRetryableError(err) {
					// Retryable error (e.g., temporary DB connection issue, timeout)
					// Don't commit offset - message will be reprocessed
					h.logger.Warn().
						Err(err).
						Str("topic", msg.Topic).
						Int32("partition", msg.Partition).
						Int64("offset", msg.Offset).
						Msg("Retryable error - message will be reprocessed")

					// Don't mark the message - it will be retried
					// Return to allow session to rebalance if needed
					continue
				}

				// Permanent error (e.g., invalid JSON, unknown topic)
				// Skip this message by committing offset to avoid getting stuck
				h.logger.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int32("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Permanent error - SKIPPING message (will be committed)")

				// TODO: Send to Dead Letter Queue (DLQ) for permanent errors
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
	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Parse subscription event
	var event domain.SubscriptionEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		// Invalid JSON is a permanent error
		return fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}

	h.logger.Debug().
		Str("event_type", event.EventType).
		Int64("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("Processing subscription event")

	// Check context again before calling handler
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Route to appropriate handler based on topic
	switch msg.Topic {
	case topicSubscriptionCreated:
		if err := h.handler.HandleSubscriptionCreated(ctx, event.UserID, event.ChannelID, event.ChannelName); err != nil {
			return err
		}
		return nil

	case topicSubscriptionDeleted:
		if err := h.handler.HandleSubscriptionDeleted(ctx, event.UserID, event.ChannelID); err != nil {
			return err
		}
		return nil

	default:
		// Unknown topic is a permanent error
		return fmt.Errorf("%w: %s", ErrUnknownTopic, msg.Topic)
	}
}

// isRetryableError determines if an error is retryable or permanent
//
// Retryable errors:
// - Context cancelled/deadline exceeded (system shutting down)
// - Any error from handler (could be temporary DB issues, network timeouts, etc.)
//
// Permanent errors:
// - Invalid message format (ErrInvalidMessage)
// - Unknown topic (ErrUnknownTopic)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are retryable (system is shutting down gracefully)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Permanent errors - don't retry
	if errors.Is(err, ErrInvalidMessage) || errors.Is(err, ErrUnknownTopic) {
		return false
	}

	// All other errors from handler are considered retryable by default
	// This is a conservative approach - better to retry than lose data
	// Handler can wrap errors with ErrInvalidMessage if they want to skip
	return true
}
