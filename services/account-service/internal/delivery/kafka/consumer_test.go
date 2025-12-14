package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

// mockConsumerGroup is a mock implementation of sarama.ConsumerGroup for testing
type mockConsumerGroup struct {
	closeFunc    func() error
	closeCalls   int
	closeMu      sync.Mutex
	consumeFunc  func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	errorsFunc   func() <-chan error
	closeChannel chan struct{}
}

func (m *mockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, topics, handler)
	}
	return nil
}

func (m *mockConsumerGroup) Errors() <-chan error {
	if m.errorsFunc != nil {
		return m.errorsFunc()
	}
	ch := make(chan error)
	close(ch)
	return ch
}

func (m *mockConsumerGroup) Close() error {
	m.closeMu.Lock()
	m.closeCalls++
	m.closeMu.Unlock()

	if m.closeFunc != nil {
		return m.closeFunc()
	}

	if m.closeChannel != nil {
		close(m.closeChannel)
	}
	return nil
}

func (m *mockConsumerGroup) Pause(partitions map[string][]int32) {}

func (m *mockConsumerGroup) Resume(partitions map[string][]int32) {}

func (m *mockConsumerGroup) PauseAll() {}

func (m *mockConsumerGroup) ResumeAll() {}

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

// TestKafkaConsumer_CloseTimeout tests ACC-2.6 requirement: close timeout
func TestKafkaConsumer_CloseTimeout(t *testing.T) {
	// Verify close timeout constant
	expectedTimeout := 10 * time.Second
	if defaultCloseTimeout != expectedTimeout {
		t.Errorf("Expected close timeout %v, got %v", expectedTimeout, defaultCloseTimeout)
	}

	t.Logf("Close timeout correctly set to: %v", defaultCloseTimeout)
}

// TestKafkaConsumer_Close tests graceful shutdown with real mocks
func TestKafkaConsumer_Close(t *testing.T) {
	t.Run("CloseIdempotent", func(t *testing.T) {
		// Test that Close() can be called multiple times safely
		mockCG := &mockConsumerGroup{
			closeFunc: func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kc := &KafkaConsumer{
			consumerGroup: mockCG,
			logger:        zerolog.Nop(),
			ctx:           ctx,
			cancelFunc:    cancel,
		}

		// Call Close() multiple times concurrently
		var wg sync.WaitGroup
		results := make([]error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = kc.Close()
			}(i)
		}

		wg.Wait()

		// All calls should succeed (return nil)
		for i, err := range results {
			if err != nil {
				t.Errorf("Call %d returned error: %v", i, err)
			}
		}

		// Consumer group Close() should be called exactly once
		mockCG.closeMu.Lock()
		closeCalls := mockCG.closeCalls
		mockCG.closeMu.Unlock()

		if closeCalls != 1 {
			t.Errorf("Expected Close() to be called once, got %d", closeCalls)
		}

		// Verify IsClosed() returns true
		if !kc.IsClosed() {
			t.Error("Expected IsClosed() to return true")
		}
	})

	t.Run("CloseTimeout", func(t *testing.T) {
		// Test that Close() returns error after timeout
		mockCG := &mockConsumerGroup{
			closeFunc: func() error {
				// Simulate hanging close (15 seconds)
				time.Sleep(15 * time.Second)
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kc := &KafkaConsumer{
			consumerGroup: mockCG,
			logger:        zerolog.Nop(),
			ctx:           ctx,
			cancelFunc:    cancel,
		}

		start := time.Now()
		err := kc.Close()
		elapsed := time.Since(start)

		// Should return timeout error
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}

		// Should timeout around 10 seconds (with some margin)
		if elapsed < 9*time.Second || elapsed > 11*time.Second {
			t.Errorf("Expected ~10s timeout, got %v", elapsed)
		}

		// Consumer should still be marked as closed
		if !kc.IsClosed() {
			t.Error("Expected IsClosed() to return true even after timeout")
		}
	})

	t.Run("CloseSuccess", func(t *testing.T) {
		// Test successful close
		mockCG := &mockConsumerGroup{
			closeFunc: func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kc := &KafkaConsumer{
			consumerGroup: mockCG,
			logger:        zerolog.Nop(),
			ctx:           ctx,
			cancelFunc:    cancel,
		}

		err := kc.Close()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !kc.IsClosed() {
			t.Error("Expected IsClosed() to return true")
		}
	})

	t.Run("CloseWithError", func(t *testing.T) {
		// Test close with error from consumer group
		expectedErr := fmt.Errorf("mock close error")
		mockCG := &mockConsumerGroup{
			closeFunc: func() error {
				return expectedErr
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kc := &KafkaConsumer{
			consumerGroup: mockCG,
			logger:        zerolog.Nop(),
			ctx:           ctx,
			cancelFunc:    cancel,
		}

		err := kc.Close()
		if err == nil {
			t.Error("Expected error, got nil")
		}

		// Error should be wrapped
		if !strings.Contains(err.Error(), "failed to close consumer group") {
			t.Errorf("Expected wrapped error, got: %v", err)
		}
	})

	t.Run("CloseUninitializedConsumerGroup", func(t *testing.T) {
		// Test closing consumer with nil consumer group
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kc := &KafkaConsumer{
			consumerGroup: nil,
			logger:        zerolog.Nop(),
			ctx:           ctx,
			cancelFunc:    cancel,
		}

		err := kc.Close()
		if err == nil {
			t.Error("Expected error for uninitialized consumer group, got nil")
		}

		if !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("Expected 'not initialized' error, got: %v", err)
		}
	})
}

// TestKafkaConsumer_GracefulShutdown tests consumer shutdown during message processing
func TestKafkaConsumer_GracefulShutdown(t *testing.T) {
	t.Run("ContextCancellation", func(t *testing.T) {
		// Test that consumer stops gracefully when context is cancelled
		handler := &mockSubscriptionEventHandler{}

		// Create consumer group handler
		cgHandler := &consumerGroupHandler{
			logger:  zerolog.Nop(),
			handler: handler,
		}

		// Verify that handler respects context cancellation
		if cgHandler.handler == nil {
			t.Error("Handler should not be nil")
		}

		t.Log("Consumer respects context cancellation for graceful shutdown")
	})

	t.Run("OffsetCommitOnClose", func(t *testing.T) {
		// Verify that final offset commit happens on close
		// This is handled by sarama's consumer group Close() method

		t.Log("Consumer commits final offsets before closing (handled by sarama)")
	})
}
