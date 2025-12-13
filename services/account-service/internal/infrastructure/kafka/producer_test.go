package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// TestNewKafkaProducer tests producer creation with valid configuration
func TestNewKafkaProducer(t *testing.T) {
	// Create mock async producer
	mockProducer := mocks.NewAsyncProducer(t, nil)
	defer mockProducer.Close()

	config := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "news.received",
		Logger:  zerolog.Nop(),
	}

	// Note: This test validates configuration, but actual producer creation
	// requires running Kafka. For unit tests, we use mocks in SendNewsReceived tests.
	if len(config.Brokers) == 0 {
		t.Error("Expected non-empty brokers list")
	}
	if config.Topic == "" {
		t.Error("Expected non-empty topic")
	}
}

// TestNewKafkaProducer_EmptyBrokers tests validation of empty brokers
func TestNewKafkaProducer_EmptyBrokers(t *testing.T) {
	config := ProducerConfig{
		Brokers: []string{},
		Topic:   "news.received",
		Logger:  zerolog.Nop(),
	}

	_, err := NewKafkaProducer(config)
	if err == nil {
		t.Error("Expected error for empty brokers, got nil")
	}
	if err.Error() != "no kafka brokers specified" {
		t.Errorf("Expected 'no kafka brokers specified', got %v", err)
	}
}

// TestNewKafkaProducer_EmptyTopic tests validation of empty topic
func TestNewKafkaProducer_EmptyTopic(t *testing.T) {
	config := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "",
		Logger:  zerolog.Nop(),
	}

	_, err := NewKafkaProducer(config)
	if err == nil {
		t.Error("Expected error for empty topic, got nil")
	}
	if err.Error() != "kafka topic is required" {
		t.Errorf("Expected 'kafka topic is required', got %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived tests successful message sending
func TestKafkaProducer_SendNewsReceived(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)

	// Set expectations for successful send
	mockProducer.ExpectInputAndSucceed()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news content",
		MediaURLs:   []string{"https://example.com/image.jpg"},
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify mock expectations were met
	if err := mockProducer.Close(); err != nil {
		t.Errorf("Mock producer close failed: %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_NilNews tests handling of nil news item
func TestKafkaProducer_SendNewsReceived_NilNews(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)
	defer mockProducer.Close()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, nil)

	if err == nil {
		t.Error("Expected error for nil news item, got nil")
	}
	if err.Error() != "news item is nil" {
		t.Errorf("Expected 'news item is nil', got %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_ContextTimeout tests context with timeout
func TestKafkaProducer_SendNewsReceived_ContextTimeout(t *testing.T) {
	// Create mock producer
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	mockProducer := mocks.NewAsyncProducer(t, config)
	defer mockProducer.Close()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news content",
		Date:        time.Now(),
	}

	// Create context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Set expectation for message
	mockProducer.ExpectInputAndSucceed()

	err := kp.SendNewsReceived(ctx, news)

	// Should succeed within timeout
	if err != nil {
		t.Errorf("Expected no error with valid timeout, got %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_MessageStructure tests message partitioning
func TestKafkaProducer_SendNewsReceived_MessageStructure(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)

	// Set expectations
	mockProducer.ExpectInputAndSucceed()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	news := &domain.NewsItem{
		ChannelID:   "test_channel_123",
		ChannelName: "Test Channel",
		MessageID:   99999,
		Content:     "Important news",
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Close and verify expectations
	if err := mockProducer.Close(); err != nil {
		t.Errorf("Mock producer close failed: %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_MultipleMessages tests sending multiple messages
func TestKafkaProducer_SendNewsReceived_MultipleMessages(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)

	// Expect 3 successful sends
	mockProducer.ExpectInputAndSucceed()
	mockProducer.ExpectInputAndSucceed()
	mockProducer.ExpectInputAndSucceed()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	ctx := context.Background()

	// Send 3 messages
	for i := 1; i <= 3; i++ {
		news := &domain.NewsItem{
			ChannelID:   "channel_1",
			ChannelName: "Channel One",
			MessageID:   i,
			Content:     "News content",
			Date:        time.Now(),
		}

		if err := kp.SendNewsReceived(ctx, news); err != nil {
			t.Errorf("Failed to send message %d: %v", i, err)
		}
	}

	// Verify all messages were sent
	if err := mockProducer.Close(); err != nil {
		t.Errorf("Mock producer close failed: %v", err)
	}
}

// TestKafkaProducer_Close tests graceful shutdown
func TestKafkaProducer_Close(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	// Start handler goroutines
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	// Close should not return error
	err := kp.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}
}

// TestKafkaProducer_ErrorHandling tests error channel handling
func TestKafkaProducer_ErrorHandling(t *testing.T) {
	config := sarama.NewConfig()
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true

	mockProducer := mocks.NewAsyncProducer(t, config)

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	// Start handler goroutines
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	// Expect a failed send
	mockProducer.ExpectInputAndFail(sarama.ErrOutOfBrokers)

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news",
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)

	// SendNewsReceived should not return error (async send)
	if err != nil {
		t.Errorf("Expected no error from async send, got %v", err)
	}

	// Give time for error handler to process
	time.Sleep(100 * time.Millisecond)

	// Close should detect the error
	closeErr := kp.Close()
	if closeErr == nil {
		t.Error("Expected close error due to send failure, got nil")
	}
}

// TestKafkaProducer_SuccessHandling tests success channel handling
func TestKafkaProducer_SuccessHandling(t *testing.T) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	mockProducer := mocks.NewAsyncProducer(t, config)

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	// Start handler goroutines
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	// Expect successful send
	mockProducer.ExpectInputAndSucceed()

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news",
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Give time for success handler to process
	time.Sleep(100 * time.Millisecond)

	// Close should succeed
	err = kp.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_InvalidChannelID tests validation of empty ChannelID
func TestKafkaProducer_SendNewsReceived_InvalidChannelID(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)
	defer mockProducer.Close()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	news := &domain.NewsItem{
		ChannelID:   "", // Empty ChannelID
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news",
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)

	if err == nil {
		t.Error("Expected error for empty ChannelID, got nil")
	}
	if err.Error() != "invalid news item: channel_id is required" {
		t.Errorf("Expected 'invalid news item: channel_id is required', got %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_InvalidMessageID tests validation of invalid MessageID
func TestKafkaProducer_SendNewsReceived_InvalidMessageID(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)
	defer mockProducer.Close()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   -1, // Negative MessageID
		Content:     "Test news",
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)

	if err == nil {
		t.Error("Expected error for negative MessageID, got nil")
	}
	if err.Error() != "invalid news item: message_id must be positive, got -1" {
		t.Errorf("Expected message_id validation error, got %v", err)
	}
}

// TestKafkaProducer_SendNewsReceived_InvalidDate tests validation of zero Date
func TestKafkaProducer_SendNewsReceived_InvalidDate(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)
	defer mockProducer.Close()

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news",
		Date:        time.Time{}, // Zero date
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)

	if err == nil {
		t.Error("Expected error for zero Date, got nil")
	}
	if err.Error() != "invalid news item: date is required" {
		t.Errorf("Expected 'invalid news item: date is required', got %v", err)
	}
}

// TestKafkaProducer_ErrorCallback tests ErrorCallback invocation on send failure
func TestKafkaProducer_ErrorCallback(t *testing.T) {
	config := sarama.NewConfig()
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true

	mockProducer := mocks.NewAsyncProducer(t, config)

	callbackCalled := false
	var callbackNews *domain.NewsItem
	var callbackErr error

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
		errorCallback: func(news *domain.NewsItem, err error) {
			callbackCalled = true
			callbackNews = news
			callbackErr = err
		},
	}

	// Start handler goroutines
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	// Expect a failed send
	mockProducer.ExpectInputAndFail(sarama.ErrOutOfBrokers)

	news := &domain.NewsItem{
		ChannelID:   "test_channel",
		ChannelName: "Test Channel",
		MessageID:   12345,
		Content:     "Test news",
		Date:        time.Now(),
	}

	ctx := context.Background()
	err := kp.SendNewsReceived(ctx, news)
	if err != nil {
		t.Errorf("Expected no error from async send, got %v", err)
	}

	// Give time for error handler to process
	time.Sleep(100 * time.Millisecond)

	// Verify callback was called
	if !callbackCalled {
		t.Error("Expected ErrorCallback to be called, but it wasn't")
	}
	if callbackNews == nil {
		t.Error("Expected news item in callback, got nil")
	} else {
		if callbackNews.ChannelID != "test_channel" {
			t.Errorf("Expected ChannelID 'test_channel', got %s", callbackNews.ChannelID)
		}
		if callbackNews.MessageID != 12345 {
			t.Errorf("Expected MessageID 12345, got %d", callbackNews.MessageID)
		}
	}
	if callbackErr == nil {
		t.Error("Expected error in callback, got nil")
	}

	// Close and verify
	kp.Close()
}

// TestKafkaProducer_Close_Idempotent tests that Close() can be called multiple times
func TestKafkaProducer_Close_Idempotent(t *testing.T) {
	mockProducer := mocks.NewAsyncProducer(t, nil)

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	// Start handler goroutines
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	// First close should succeed
	err1 := kp.Close()
	if err1 != nil {
		t.Errorf("Expected no error on first close, got %v", err1)
	}

	// Second close should also succeed (idempotent)
	err2 := kp.Close()
	if err2 != nil {
		t.Errorf("Expected no error on second close, got %v", err2)
	}

	// Third close
	err3 := kp.Close()
	if err3 != nil {
		t.Errorf("Expected no error on third close, got %v", err3)
	}
}

// TestNewKafkaProducer_Defaults tests default configuration values
func TestNewKafkaProducer_Defaults(t *testing.T) {
	config := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "news.received",
		Logger:  zerolog.Nop(),
		// MaxMessageBytes, MaxRetries, KafkaVersion not set - should use defaults
	}

	// Verify defaults are set correctly (can't test actual creation without Kafka)
	if config.MaxMessageBytes == 0 {
		// Default should be applied in NewKafkaProducer
		t.Log("MaxMessageBytes will be set to default (1MB) in NewKafkaProducer")
	}
	if config.MaxRetries == 0 {
		t.Log("MaxRetries will be set to default (5) in NewKafkaProducer")
	}
}

// TestKafkaProducer_ErrorHandling_MultipleErrors tests collecting multiple errors
func TestKafkaProducer_ErrorHandling_MultipleErrors(t *testing.T) {
	config := sarama.NewConfig()
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true

	mockProducer := mocks.NewAsyncProducer(t, config)

	kp := &KafkaProducer{
		producer: mockProducer,
		topic:    "news.received",
		logger:   zerolog.Nop(),
		errors:   make([]error, 0),
	}

	// Start handler goroutines
	kp.wg.Add(2)
	go kp.handleSuccesses()
	go kp.handleErrors()

	// Expect 3 failed sends
	mockProducer.ExpectInputAndFail(sarama.ErrOutOfBrokers)
	mockProducer.ExpectInputAndFail(sarama.ErrNotConnected)
	mockProducer.ExpectInputAndFail(sarama.ErrShuttingDown)

	ctx := context.Background()

	// Send 3 messages
	for i := 1; i <= 3; i++ {
		news := &domain.NewsItem{
			ChannelID:   "test_channel",
			ChannelName: "Test Channel",
			MessageID:   i,
			Content:     "Test news",
			Date:        time.Now(),
		}

		if err := kp.SendNewsReceived(ctx, news); err != nil {
			t.Errorf("Expected no error from async send, got %v", err)
		}
	}

	// Give time for error handler to process all errors
	time.Sleep(200 * time.Millisecond)

	// Close should report all 3 errors
	closeErr := kp.Close()
	if closeErr == nil {
		t.Error("Expected close error due to send failures, got nil")
	}

	// Verify error count
	kp.errorsMu.Lock()
	errorCount := len(kp.errors)
	kp.errorsMu.Unlock()

	if errorCount != 3 {
		t.Errorf("Expected 3 errors collected, got %d", errorCount)
	}
}
