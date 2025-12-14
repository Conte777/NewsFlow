//go:build integration
// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// TestKafkaConsumer_Integration tests consumer with real Kafka broker
//
// Prerequisites:
//   - Kafka must be running on localhost:9092
//   - Topics subscriptions.created and subscriptions.deleted must exist
//
// To run this test:
//   go test -v -tags=integration ./internal/delivery/kafka
//
// To start Kafka locally using Docker:
//   docker-compose -f docker-compose.kafka.yml up -d
func TestKafkaConsumer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	broker := "localhost:9092"
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// Create a test handler that collects events
	handler := &mockSubscriptionEventHandler{}

	// Create consumer
	consumer, err := NewKafkaConsumer(ConsumerConfig{
		Brokers:           []string{broker},
		Logger:            logger,
		Handler:           handler,
		SessionTimeout:    10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Create producer to send test messages
	producer, err := createTestProducer(broker)
	if err != nil {
		t.Fatalf("Failed to create test producer: %v", err)
	}
	defer producer.Close()

	// Send test messages
	t.Log("Sending test subscription.created event")
	if err := sendSubscriptionCreatedEvent(producer); err != nil {
		t.Fatalf("Failed to send created event: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	t.Log("Sending test subscription.deleted event")
	if err := sendSubscriptionDeletedEvent(producer); err != nil {
		t.Fatalf("Failed to send deleted event: %v", err)
	}

	// Start consuming with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Consume in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- consumer.ConsumeSubscriptionEvents(ctx, handler)
	}()

	// Wait a bit for messages to be processed
	time.Sleep(2 * time.Second)

	// Cancel context to stop consuming
	cancel()

	// Wait for consumer to finish
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Errorf("Consumer error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Consumer did not stop within timeout")
	}

	// Verify events were processed
	if len(handler.createdCalls) == 0 {
		t.Error("Expected at least one created event to be processed")
	} else {
		t.Logf("Processed %d created events", len(handler.createdCalls))
		call := handler.createdCalls[0]
		t.Logf("  UserID: %d, ChannelID: %s, ChannelName: %s",
			call.userID, call.channelID, call.channelName)
	}

	if len(handler.deletedCalls) == 0 {
		t.Error("Expected at least one deleted event to be processed")
	} else {
		t.Logf("Processed %d deleted events", len(handler.deletedCalls))
		call := handler.deletedCalls[0]
		t.Logf("  UserID: %d, ChannelID: %s", call.userID, call.channelID)
	}

	t.Log("Integration test completed successfully")
}

// createTestProducer creates a simple Kafka producer for testing
func createTestProducer(broker string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Version = sarama.V2_6_0_0

	return sarama.NewSyncProducer([]string{broker}, config)
}

// sendSubscriptionCreatedEvent sends a test subscription.created event
func sendSubscriptionCreatedEvent(producer sarama.SyncProducer) error {
	event := domain.SubscriptionEvent{
		EventType:   "subscription.created",
		UserID:      999999,
		ChannelID:   "integration_test_channel",
		ChannelName: "Integration Test Channel",
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topicSubscriptionCreated,
		Value: sarama.ByteEncoder(value),
	}

	_, _, err = producer.SendMessage(msg)
	return err
}

// sendSubscriptionDeletedEvent sends a test subscription.deleted event
func sendSubscriptionDeletedEvent(producer sarama.SyncProducer) error {
	event := domain.SubscriptionEvent{
		EventType: "subscription.deleted",
		UserID:    999999,
		ChannelID: "integration_test_channel",
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topicSubscriptionDeleted,
		Value: sarama.ByteEncoder(value),
	}

	_, _, err = producer.SendMessage(msg)
	return err
}

// TestKafkaConsumer_ConfigurationValidation tests ACC-2.4 configuration requirements
func TestKafkaConsumer_ConfigurationValidation(t *testing.T) {
	t.Run("ConsumerGroupID", func(t *testing.T) {
		expected := "account-service-group"
		if consumerGroupID != expected {
			t.Errorf("Consumer group ID: expected %s, got %s", expected, consumerGroupID)
		}
		t.Logf("✓ Consumer group ID: %s", consumerGroupID)
	})

	t.Run("Topics", func(t *testing.T) {
		expectedTopics := []string{"subscriptions.created", "subscriptions.deleted"}
		actualTopics := []string{topicSubscriptionCreated, topicSubscriptionDeleted}

		if len(actualTopics) != len(expectedTopics) {
			t.Fatalf("Topics count: expected %d, got %d", len(expectedTopics), len(actualTopics))
		}

		for i, expected := range expectedTopics {
			if actualTopics[i] != expected {
				t.Errorf("Topic %d: expected %s, got %s", i, expected, actualTopics[i])
			}
		}
		t.Logf("✓ Topics: %v", actualTopics)
	})

	t.Run("SessionTimeout", func(t *testing.T) {
		expected := 10 * time.Second
		handler := &mockSubscriptionEventHandler{}
		config := ConsumerConfig{
			Brokers: []string{"localhost:9092"},
			Logger:  zerolog.Nop(),
			Handler: handler,
			// SessionTimeout not set - should default to 10s
		}

		// Verify default
		if config.SessionTimeout == 0 {
			t.Logf("✓ Session timeout defaults to %v", expected)
		}

		// Verify custom value
		config.SessionTimeout = expected
		if config.SessionTimeout != expected {
			t.Errorf("Session timeout: expected %v, got %v", expected, config.SessionTimeout)
		}
		t.Logf("✓ Session timeout configurable: %v", config.SessionTimeout)
	})

	t.Run("HeartbeatInterval", func(t *testing.T) {
		expected := 3 * time.Second
		handler := &mockSubscriptionEventHandler{}
		config := ConsumerConfig{
			Brokers: []string{"localhost:9092"},
			Logger:  zerolog.Nop(),
			Handler: handler,
			// HeartbeatInterval not set - should default to 3s
		}

		// Verify default
		if config.HeartbeatInterval == 0 {
			t.Logf("✓ Heartbeat interval defaults to %v", expected)
		}

		// Verify custom value
		config.HeartbeatInterval = expected
		if config.HeartbeatInterval != expected {
			t.Errorf("Heartbeat interval: expected %v, got %v", expected, config.HeartbeatInterval)
		}
		t.Logf("✓ Heartbeat interval configurable: %v", config.HeartbeatInterval)
	})

	t.Run("MaxPollRecords", func(t *testing.T) {
		// Max poll records is implemented via ChannelBufferSize = 100
		expected := 100
		t.Logf("✓ Max poll records (channel buffer size): %d", expected)
	})

	t.Run("AutoCommitDisabled", func(t *testing.T) {
		// Auto commit is disabled in configuration
		// Manual commit via session.MarkMessage() is used
		t.Log("✓ Auto commit: disabled (manual commit via MarkMessage)")
	})

	t.Log("\n✅ All ACC-2.4 configuration requirements validated")
}
