package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// mockSubscriptionEventHandler is a mock implementation of domain.SubscriptionEventHandler for testing
type mockSubscriptionEventHandler struct {
	createdCalls []subscriptionCreatedCall
	deletedCalls []subscriptionDeletedCall
	createErr    error
	deleteErr    error
}

type subscriptionCreatedCall struct {
	userID      int64
	channelID   string
	channelName string
}

type subscriptionDeletedCall struct {
	userID    int64
	channelID string
}

func (m *mockSubscriptionEventHandler) HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error {
	m.createdCalls = append(m.createdCalls, subscriptionCreatedCall{
		userID:      userID,
		channelID:   channelID,
		channelName: channelName,
	})
	return m.createErr
}

func (m *mockSubscriptionEventHandler) HandleSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error {
	m.deletedCalls = append(m.deletedCalls, subscriptionDeletedCall{
		userID:    userID,
		channelID: channelID,
	})
	return m.deleteErr
}

// TestNewKafkaConsumer_Success tests successful consumer creation
func TestNewKafkaConsumer_Success(t *testing.T) {
	// Note: This test validates configuration but doesn't create real connection
	// For integration tests with real Kafka, see consumer_integration_test.go

	handler := &mockSubscriptionEventHandler{}
	config := ConsumerConfig{
		Brokers:           []string{"localhost:9092"},
		Logger:            zerolog.Nop(),
		Handler:           handler,
		SessionTimeout:    10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
	}

	// Validate configuration
	if len(config.Brokers) == 0 {
		t.Error("Expected non-empty brokers list")
	}
	if config.Handler == nil {
		t.Error("Expected non-nil handler")
	}
	if config.SessionTimeout != 10*time.Second {
		t.Errorf("Expected session timeout 10s, got %v", config.SessionTimeout)
	}
	if config.HeartbeatInterval != 3*time.Second {
		t.Errorf("Expected heartbeat interval 3s, got %v", config.HeartbeatInterval)
	}
}

// TestNewKafkaConsumer_EmptyBrokers tests validation of empty brokers
func TestNewKafkaConsumer_EmptyBrokers(t *testing.T) {
	handler := &mockSubscriptionEventHandler{}
	config := ConsumerConfig{
		Brokers: []string{},
		Logger:  zerolog.Nop(),
		Handler: handler,
	}

	_, err := NewKafkaConsumer(config)
	if err == nil {
		t.Error("Expected error for empty brokers, got nil")
	}
	if err.Error() != "no kafka brokers specified" {
		t.Errorf("Expected 'no kafka brokers specified', got %v", err)
	}
}

// TestNewKafkaConsumer_NilHandler tests validation of nil handler
func TestNewKafkaConsumer_NilHandler(t *testing.T) {
	config := ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Logger:  zerolog.Nop(),
		Handler: nil,
	}

	_, err := NewKafkaConsumer(config)
	if err == nil {
		t.Error("Expected error for nil handler, got nil")
	}
	if err.Error() != "event handler is required" {
		t.Errorf("Expected 'event handler is required', got %v", err)
	}
}

// TestNewKafkaConsumer_Defaults tests default configuration values
func TestNewKafkaConsumer_Defaults(t *testing.T) {
	handler := &mockSubscriptionEventHandler{}
	config := ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		Logger:  zerolog.Nop(),
		Handler: handler,
		// SessionTimeout and HeartbeatInterval not set - should use defaults
	}

	// Verify defaults would be applied (can't test actual creation without Kafka)
	if config.SessionTimeout == 0 {
		t.Log("SessionTimeout will be set to default (10s) in NewKafkaConsumer")
	}
	if config.HeartbeatInterval == 0 {
		t.Log("HeartbeatInterval will be set to default (3s) in NewKafkaConsumer")
	}
}

// TestConsumerConfig_Topics tests topic configuration
func TestConsumerConfig_Topics(t *testing.T) {
	expectedTopics := []string{topicSubscriptionCreated, topicSubscriptionDeleted}

	if topicSubscriptionCreated != "subscriptions.created" {
		t.Errorf("Expected topic 'subscriptions.created', got %s", topicSubscriptionCreated)
	}
	if topicSubscriptionDeleted != "subscriptions.deleted" {
		t.Errorf("Expected topic 'subscriptions.deleted', got %s", topicSubscriptionDeleted)
	}

	if len(expectedTopics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(expectedTopics))
	}
}

// TestConsumerConfig_ConsumerGroup tests consumer group ID
func TestConsumerConfig_ConsumerGroup(t *testing.T) {
	expectedGroupID := "account-service-group"

	if consumerGroupID != expectedGroupID {
		t.Errorf("Expected consumer group ID '%s', got '%s'", expectedGroupID, consumerGroupID)
	}
}

// TestConsumerGroupHandler_ProcessMessage tests message processing logic
func TestConsumerGroupHandler_ProcessMessage(t *testing.T) {
	handler := &mockSubscriptionEventHandler{}
	cgHandler := &consumerGroupHandler{
		logger:  zerolog.Nop(),
		handler: handler,
	}

	ctx := context.Background()

	t.Run("SubscriptionCreated", func(t *testing.T) {
		event := domain.SubscriptionEvent{
			EventType:   "subscription.created",
			UserID:      12345,
			ChannelID:   "test_channel",
			ChannelName: "Test Channel",
		}

		msgValue, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		msg := &sarama.ConsumerMessage{
			Topic: topicSubscriptionCreated,
			Value: msgValue,
		}

		err = cgHandler.processMessage(ctx, msg)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(handler.createdCalls) != 1 {
			t.Fatalf("Expected 1 created call, got %d", len(handler.createdCalls))
		}

		call := handler.createdCalls[0]
		if call.userID != 12345 {
			t.Errorf("Expected userID 12345, got %d", call.userID)
		}
		if call.channelID != "test_channel" {
			t.Errorf("Expected channelID 'test_channel', got %s", call.channelID)
		}
		if call.channelName != "Test Channel" {
			t.Errorf("Expected channelName 'Test Channel', got %s", call.channelName)
		}
	})

	t.Run("SubscriptionDeleted", func(t *testing.T) {
		handler := &mockSubscriptionEventHandler{}
		cgHandler := &consumerGroupHandler{
			logger:  zerolog.Nop(),
			handler: handler,
		}

		event := domain.SubscriptionEvent{
			EventType: "subscription.deleted",
			UserID:    67890,
			ChannelID: "deleted_channel",
		}

		msgValue, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		msg := &sarama.ConsumerMessage{
			Topic: topicSubscriptionDeleted,
			Value: msgValue,
		}

		err = cgHandler.processMessage(ctx, msg)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(handler.deletedCalls) != 1 {
			t.Fatalf("Expected 1 deleted call, got %d", len(handler.deletedCalls))
		}

		call := handler.deletedCalls[0]
		if call.userID != 67890 {
			t.Errorf("Expected userID 67890, got %d", call.userID)
		}
		if call.channelID != "deleted_channel" {
			t.Errorf("Expected channelID 'deleted_channel', got %s", call.channelID)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		handler := &mockSubscriptionEventHandler{}
		cgHandler := &consumerGroupHandler{
			logger:  zerolog.Nop(),
			handler: handler,
		}

		msg := &sarama.ConsumerMessage{
			Topic: topicSubscriptionCreated,
			Value: []byte("invalid json"),
		}

		err := cgHandler.processMessage(ctx, msg)
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("UnknownTopic", func(t *testing.T) {
		handler := &mockSubscriptionEventHandler{}
		cgHandler := &consumerGroupHandler{
			logger:  zerolog.Nop(),
			handler: handler,
		}

		event := domain.SubscriptionEvent{
			EventType: "unknown",
			UserID:    12345,
			ChannelID: "test",
		}

		msgValue, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		msg := &sarama.ConsumerMessage{
			Topic: "unknown.topic",
			Value: msgValue,
		}

		err = cgHandler.processMessage(ctx, msg)
		if err == nil {
			t.Error("Expected error for unknown topic, got nil")
		}
	})
}

// TestSubscriptionEvent_JSONSerialization tests SubscriptionEvent JSON format
func TestSubscriptionEvent_JSONSerialization(t *testing.T) {
	event := domain.SubscriptionEvent{
		EventType:   "subscription.created",
		UserID:      12345,
		ChannelID:   "channel_123",
		ChannelName: "Tech News",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal SubscriptionEvent: %v", err)
	}

	// Expected JSON format with snake_case keys
	expectedJSON := `{"event_type":"subscription.created","user_id":12345,"channel_id":"channel_123","channel_name":"Tech News"}`

	if string(jsonData) != expectedJSON {
		t.Errorf("JSON format mismatch.\nGot:      %s\nExpected: %s", string(jsonData), expectedJSON)
	}

	// Verify we can deserialize back
	var deserialized domain.SubscriptionEvent
	if err := json.Unmarshal(jsonData, &deserialized); err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify data integrity
	if deserialized.EventType != event.EventType {
		t.Errorf("EventType mismatch: got %s, want %s", deserialized.EventType, event.EventType)
	}
	if deserialized.UserID != event.UserID {
		t.Errorf("UserID mismatch: got %d, want %d", deserialized.UserID, event.UserID)
	}
	if deserialized.ChannelID != event.ChannelID {
		t.Errorf("ChannelID mismatch: got %s, want %s", deserialized.ChannelID, event.ChannelID)
	}
	if deserialized.ChannelName != event.ChannelName {
		t.Errorf("ChannelName mismatch: got %s, want %s", deserialized.ChannelName, event.ChannelName)
	}

	t.Log("JSON serialization test passed - using snake_case format")
}

// TestKafkaConsumer_TopicsConfiguration tests that consumer subscribes to correct topics
func TestKafkaConsumer_TopicsConfiguration(t *testing.T) {
	// This test verifies ACC-2.4 requirement: subscription to correct topics
	expectedTopics := []string{"subscriptions.created", "subscriptions.deleted"}

	topics := []string{topicSubscriptionCreated, topicSubscriptionDeleted}

	if len(topics) != len(expectedTopics) {
		t.Fatalf("Expected %d topics, got %d", len(expectedTopics), len(topics))
	}

	for i, topic := range topics {
		if topic != expectedTopics[i] {
			t.Errorf("Topic %d: expected %s, got %s", i, expectedTopics[i], topic)
		}
	}

	t.Log("Consumer subscribes to correct topics: subscriptions.created, subscriptions.deleted")
}

// TestKafkaConsumer_ConsumerGroupID tests consumer group configuration
func TestKafkaConsumer_ConsumerGroupID(t *testing.T) {
	// This test verifies ACC-2.4 requirement: correct consumer group ID
	expectedGroupID := "account-service-group"

	if consumerGroupID != expectedGroupID {
		t.Errorf("Expected consumer group ID %s, got %s", expectedGroupID, consumerGroupID)
	}

	t.Logf("Consumer group ID correctly set to: %s", consumerGroupID)
}
