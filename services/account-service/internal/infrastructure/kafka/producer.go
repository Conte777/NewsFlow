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
	// maxStoredErrors is the maximum number of errors to keep in memory
	// This prevents unbounded memory growth during long-running operations
	maxStoredErrors = 100
)

// ErrorCallback is called when a message fails to send
// Can be used to implement retry logic, dead-letter queue, or alerting
type ErrorCallback func(news *domain.NewsItem, err error)

// KafkaProducer sends news items to Kafka using asynchronous producer
type KafkaProducer struct {
	producer      sarama.AsyncProducer
	topic         string
	logger        zerolog.Logger
	errorCallback ErrorCallback
	wg            sync.WaitGroup
	closeOnce     sync.Once
	closeErr      error
	errors        []error // Collect all errors during operation
	errorsMu      sync.Mutex
}

// ProducerConfig holds configuration for Kafka producer
type ProducerConfig struct {
	Brokers         []string       // Kafka broker addresses
	Topic           string         // Topic name for news items
	Logger          zerolog.Logger // Logger for monitoring
	ErrorCallback   ErrorCallback  // Optional callback for handling send errors
	MaxMessageBytes int            // Max message size in bytes (default: 1MB)
	MaxRetries      int            // Max retries for failed sends (default: 5)
}

// ValidateBrokers checks if Kafka brokers are accessible
// Returns error if cannot connect to any broker
func ValidateBrokers(brokers []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no brokers specified")
	}

	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka brokers: %w", err)
	}
	defer client.Close()

	// Check if we can communicate with brokers
	if err := client.RefreshMetadata(); err != nil {
		return fmt.Errorf("failed to refresh metadata from Kafka: %w", err)
	}

	return nil
}

// NewKafkaProducer creates a new Kafka producer with async producer configuration
//
// Configuration highlights:
// - Asynchronous producer for high throughput
// - Snappy compression for bandwidth optimization
// - Idempotent mode for at-least-once delivery with deduplication
// - Hash partitioner based on channel_id for ordering guarantees
func NewKafkaProducer(cfg ProducerConfig) (domain.KafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("no kafka brokers specified")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("kafka topic is required")
	}

	// Set defaults for optional config values
	if cfg.MaxMessageBytes <= 0 {
		cfg.MaxMessageBytes = 1000000 // 1MB default
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5 // 5 retries default
	}

	config := sarama.NewConfig()

	// Producer settings for high performance and reliability
	config.Producer.Return.Successes = true // Required for async producer monitoring
	config.Producer.Return.Errors = true    // Required for error handling

	// Compression: Snappy (good balance between speed and compression ratio)
	config.Producer.Compression = sarama.CompressionSnappy

	// Idempotent mode: ensures at-least-once delivery with automatic deduplication
	// Note: This is NOT exactly-once semantics, which requires transactions
	config.Producer.Idempotent = true
	config.Producer.RequiredAcks = sarama.WaitForAll // Required for idempotent producer
	config.Net.MaxOpenRequests = 1                   // Required for idempotent producer
	config.Producer.MaxMessageBytes = cfg.MaxMessageBytes
	config.Producer.Retry.Max = cfg.MaxRetries

	// Partitioner: hash by channel_id for message ordering per channel
	config.Producer.Partitioner = sarama.NewHashPartitioner

	// Set client ID for identification
	config.ClientID = "account-service-producer"

	// Kafka version compatibility (using stable version)
	config.Version = sarama.V2_6_0_0

	// Create async producer
	producer, err := sarama.NewAsyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	kp := &KafkaProducer{
		producer:      producer,
		topic:         cfg.Topic,
		logger:        cfg.Logger,
		errorCallback: cfg.ErrorCallback,
		errors:        make([]error, 0),
	}

	// Start goroutines to handle async responses
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	cfg.Logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.Topic).
		Int("max_message_bytes", cfg.MaxMessageBytes).
		Int("max_retries", cfg.MaxRetries).
		Msg("Kafka producer initialized successfully")

	return kp, nil
}

// SendNewsReceived sends a news item to Kafka asynchronously
//
// The method validates NewsItem, marshals it to JSON, and sends it to Kafka topic.
// Uses channel_id as the partition key to ensure message ordering per channel.
//
// Returns error if validation fails, context is cancelled, or encoding fails.
// Actual Kafka send errors are handled asynchronously via error channel and ErrorCallback.
func (p *KafkaProducer) SendNewsReceived(ctx context.Context, news *domain.NewsItem) error {
	if news == nil {
		return fmt.Errorf("news item is nil")
	}

	// Validate NewsItem fields
	if err := validateNewsItem(news); err != nil {
		return fmt.Errorf("invalid news item: %w", err)
	}

	// Check context before expensive operations
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before sending: %w", ctx.Err())
	default:
	}

	// Marshal news item to JSON
	value, err := json.Marshal(news)
	if err != nil {
		return fmt.Errorf("failed to marshal news item: %w", err)
	}

	// Create Kafka message with channel_id as key (for hash partitioning)
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(news.ChannelID), // Partition by channel_id
		Value: sarama.ByteEncoder(value),
		Timestamp: news.Date, // Use original message timestamp
	}

	// Send message asynchronously
	select {
	case p.producer.Input() <- msg:
		p.logger.Debug().
			Str("channel_id", news.ChannelID).
			Int("message_id", news.MessageID).
			Msg("News item queued for sending to Kafka")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while sending message: %w", ctx.Err())
	}
}

// validateNewsItem validates NewsItem fields
func validateNewsItem(news *domain.NewsItem) error {
	if news.ChannelID == "" {
		return fmt.Errorf("channel_id is required")
	}
	if news.MessageID <= 0 {
		return fmt.Errorf("message_id must be positive, got %d", news.MessageID)
	}
	if news.Date.IsZero() {
		return fmt.Errorf("date is required")
	}
	return nil
}

// handleSuccesses processes successfully sent messages
//
// Runs in a separate goroutine and logs successful message deliveries.
func (p *KafkaProducer) handleSuccesses() {
	defer p.wg.Done()

	for msg := range p.producer.Successes() {
		p.logger.Debug().
			Str("topic", msg.Topic).
			Int32("partition", msg.Partition).
			Int64("offset", msg.Offset).
			Msg("Message sent to Kafka successfully")
	}

	p.logger.Info().Msg("Success handler stopped")
}

// handleErrors processes failed message deliveries
//
// Runs in a separate goroutine and logs errors.
// Calls ErrorCallback if configured for custom error handling (e.g., DLQ, retry).
func (p *KafkaProducer) handleErrors() {
	defer p.wg.Done()

	for producerErr := range p.producer.Errors() {
		p.logger.Error().
			Err(producerErr.Err).
			Str("topic", producerErr.Msg.Topic).
			Interface("key", producerErr.Msg.Key).
			Msg("Failed to send message to Kafka")

		// Collect errors for Close() method (with size limit to prevent memory leak)
		p.errorsMu.Lock()
		if len(p.errors) < maxStoredErrors {
			p.errors = append(p.errors, producerErr.Err)
		} else if len(p.errors) == maxStoredErrors {
			// Log warning only once when limit is reached
			p.logger.Warn().
				Int("max_errors", maxStoredErrors).
				Msg("Maximum stored errors limit reached, subsequent errors will be dropped")
			p.errors = append(p.errors, fmt.Errorf("max errors limit reached, subsequent errors dropped"))
		}
		p.errorsMu.Unlock()

		// Call error callback if configured
		if p.errorCallback != nil {
			// Try to unmarshal the failed message to NewsItem
			var news domain.NewsItem
			if msgBytes, ok := producerErr.Msg.Value.(sarama.ByteEncoder); ok {
				if err := json.Unmarshal([]byte(msgBytes), &news); err == nil {
					p.errorCallback(&news, producerErr.Err)
				} else {
					p.logger.Warn().Err(err).Msg("Failed to unmarshal error message for callback")
				}
			}
		}
	}

	p.logger.Info().Msg("Error handler stopped")
}

// Close gracefully shuts down the Kafka producer with a default 10-second timeout
//
// The method:
// 1. Closes the producer (stops accepting new messages)
// 2. Waits for all pending messages to be sent (with timeout)
// 3. Waits for handler goroutines to finish
// 4. Returns any errors that occurred during message delivery
//
// Close is idempotent - can be called multiple times safely.
// Uses a default timeout of 10 seconds as specified in ACC-2.3.
func (p *KafkaProducer) Close() error {
	return p.CloseWithTimeout(10 * time.Second)
}

// CloseWithTimeout gracefully shuts down the Kafka producer with a custom timeout
//
// This method allows specifying a custom timeout for waiting for pending messages to be flushed.
// If the timeout is reached before all messages are processed, it returns an error.
//
// Parameters:
//   - timeout: Maximum time to wait for pending messages to be flushed and handlers to finish
//
// Returns:
//   - error if producer close fails, timeout occurs, or errors occurred during operation
func (p *KafkaProducer) CloseWithTimeout(timeout time.Duration) error {
	p.closeOnce.Do(func() {
		p.logger.Info().
			Dur("timeout", timeout).
			Msg("Closing Kafka producer")

		var errs []error

		// Close producer - this will close Input, Successes, and Errors channels
		// The producer will flush all pending messages before closing
		if err := p.producer.Close(); err != nil {
			p.logger.Error().Err(err).Msg("Error closing Kafka producer")
			errs = append(errs, fmt.Errorf("producer close failed: %w", err))
		}

		// Wait for handler goroutines to finish processing remaining messages
		// Use a channel to implement timeout on wg.Wait()
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		// Wait with timeout
		select {
		case <-done:
			// All handlers finished successfully
			p.logger.Debug().Msg("All handler goroutines finished")

		case <-time.After(timeout):
			// Timeout reached - handlers didn't finish in time
			// Note: The goroutine with wg.Wait() will eventually finish when handlers complete,
			// as producer.Close() closes the channels they're reading from
			p.logger.Error().
				Dur("timeout", timeout).
				Msg("Timeout waiting for handlers to finish")
			errs = append(errs, fmt.Errorf("close timeout after %s: handlers did not finish in time", timeout))
		}

		// Check if any errors occurred during operation
		p.errorsMu.Lock()
		errorCount := len(p.errors)
		p.errorsMu.Unlock()

		if errorCount > 0 {
			p.logger.Warn().
				Int("error_count", errorCount).
				Msg("Kafka producer closed with errors")
			errs = append(errs, fmt.Errorf("producer had %d send errors during operation", errorCount))
		}

		// Combine all errors
		if len(errs) > 0 {
			if len(errs) == 1 {
				p.closeErr = errs[0]
			} else {
				// Multiple errors - combine them
				errMsg := "multiple errors during close:"
				for i, err := range errs {
					errMsg += fmt.Sprintf(" [%d] %v;", i+1, err)
				}
				p.closeErr = fmt.Errorf("%s", errMsg)
			}
			p.logger.Error().Err(p.closeErr).Msg("Kafka producer closed with errors")
		} else {
			p.logger.Info().Msg("Kafka producer closed successfully")
		}
	})

	return p.closeErr
}
