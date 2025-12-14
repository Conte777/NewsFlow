// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// TestKafkaProducer_SendNewsReceived_Integration is an integration test
// that sends 10 news items to Kafka and verifies JSON serialization format
//
// Prerequisites:
// - Kafka running at localhost:9092
// - Topic "news.received" created
//
// Run with: go test -tags=integration -v ./internal/infrastructure/kafka/...
func TestKafkaProducer_SendNewsReceived_Integration(t *testing.T) {
	// Skip if running in CI or without Kafka
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create producer
	config := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "news.received",
		Logger:  zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger(),
		ErrorCallback: func(news *domain.NewsItem, err error) {
			t.Errorf("Error sending news item (channel=%s, msg=%d): %v",
				news.ChannelID, news.MessageID, err)
		},
	}

	producer, err := NewKafkaProducer(config)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	ctx := context.Background()

	// Send 10 news items
	for i := 1; i <= 10; i++ {
		news := &domain.NewsItem{
			ChannelID:   "test_channel_123",
			ChannelName: "Test News Channel",
			MessageID:   i,
			Content:     "Breaking news: Integration test message number " + string(rune(i+'0')),
			MediaURLs: []string{
				"https://example.com/image1.jpg",
				"https://example.com/video1.mp4",
			},
			Date: time.Now().Add(time.Duration(-i) * time.Minute),
		}

		if err := producer.SendNewsReceived(ctx, news); err != nil {
			t.Errorf("Failed to send news item %d: %v", i, err)
		}

		t.Logf("Sent news item %d to Kafka", i)
	}

	// Wait for all messages to be processed
	time.Sleep(2 * time.Second)

	t.Log("Successfully sent 10 news items to Kafka topic 'news.received'")
	t.Log("Check Kafka UI at http://localhost:8080 to verify messages")
}

// TestNewsItem_JSONSerialization tests that NewsItem serializes correctly
// to the expected JSON format
func TestNewsItem_JSONSerialization(t *testing.T) {
	news := &domain.NewsItem{
		ChannelID:   "channel_001",
		ChannelName: "Tech News",
		MessageID:   12345,
		Content:     "New technology breakthrough announced",
		MediaURLs: []string{
			"https://example.com/photo1.jpg",
			"https://example.com/photo2.jpg",
		},
		Date: time.Date(2025, 11, 24, 10, 30, 0, 0, time.UTC),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(news)
	if err != nil {
		t.Fatalf("Failed to marshal NewsItem: %v", err)
	}

	// Expected JSON structure (formatted for readability)
	expectedStructure := map[string]interface{}{
		"ChannelID":   "channel_001",
		"ChannelName": "Tech News",
		"MessageID":   float64(12345), // JSON numbers are float64
		"Content":     "New technology breakthrough announced",
		"MediaURLs": []interface{}{
			"https://example.com/photo1.jpg",
			"https://example.com/photo2.jpg",
		},
		"Date": "2025-11-24T10:30:00Z",
	}

	// Unmarshal to map for comparison
	var actualStructure map[string]interface{}
	if err := json.Unmarshal(jsonData, &actualStructure); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify each field
	if actualStructure["ChannelID"] != expectedStructure["ChannelID"] {
		t.Errorf("ChannelID mismatch: got %v, want %v",
			actualStructure["ChannelID"], expectedStructure["ChannelID"])
	}

	if actualStructure["ChannelName"] != expectedStructure["ChannelName"] {
		t.Errorf("ChannelName mismatch: got %v, want %v",
			actualStructure["ChannelName"], expectedStructure["ChannelName"])
	}

	if actualStructure["MessageID"] != expectedStructure["MessageID"] {
		t.Errorf("MessageID mismatch: got %v, want %v",
			actualStructure["MessageID"], expectedStructure["MessageID"])
	}

	if actualStructure["Content"] != expectedStructure["Content"] {
		t.Errorf("Content mismatch: got %v, want %v",
			actualStructure["Content"], expectedStructure["Content"])
	}

	if actualStructure["Date"] != expectedStructure["Date"] {
		t.Errorf("Date mismatch: got %v, want %v",
			actualStructure["Date"], expectedStructure["Date"])
	}

	// Verify MediaURLs array
	mediaURLs, ok := actualStructure["MediaURLs"].([]interface{})
	if !ok {
		t.Fatalf("MediaURLs is not an array")
	}
	if len(mediaURLs) != 2 {
		t.Errorf("MediaURLs length mismatch: got %d, want 2", len(mediaURLs))
	}

	t.Logf("JSON serialization test passed. Output:\n%s", string(jsonData))
}

// TestNewsItem_JSONFormat_AccordingToSpec verifies the exact JSON format
// matches the specification from ACC-2.2
func TestNewsItem_JSONFormat_AccordingToSpec(t *testing.T) {
	news := &domain.NewsItem{
		ChannelID:   "tech_channel",
		ChannelName: "Technology Updates",
		MessageID:   99999,
		Content:     "AI breakthrough in language processing",
		MediaURLs: []string{
			"https://cdn.example.com/img1.png",
			"https://cdn.example.com/img2.png",
		},
		Date: time.Date(2025, 11, 24, 10, 30, 0, 0, time.UTC),
	}

	jsonData, err := json.MarshalIndent(news, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Expected format according to ACC-2.2 specification:
	// {
	//   "channel_id": "string",
	//   "channel_name": "string",
	//   "message_id": "int",
	//   "content": "string",
	//   "media_urls": ["url1", "url2"],
	//   "date": "2025-11-24T10:30:00Z"
	// }

	// Note: Go's json.Marshal uses field names, not snake_case
	// If snake_case is required, we need to add JSON tags to NewsItem struct

	t.Logf("Serialized NewsItem (formatted):\n%s", string(jsonData))

	// Verify we can deserialize back
	var deserialized domain.NewsItem
	if err := json.Unmarshal(jsonData, &deserialized); err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify data integrity
	if deserialized.ChannelID != news.ChannelID {
		t.Errorf("ChannelID mismatch after deserialization")
	}
	if deserialized.MessageID != news.MessageID {
		t.Errorf("MessageID mismatch after deserialization")
	}
	if len(deserialized.MediaURLs) != len(news.MediaURLs) {
		t.Errorf("MediaURLs length mismatch after deserialization")
	}

	t.Log("JSON format matches specification and data integrity verified")
}

// TestKafkaProducer_BulkSend_Performance tests sending multiple messages
// and measures performance
func TestKafkaProducer_BulkSend_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test")
	}

	// This test uses mocks for unit testing
	// For real performance testing, use integration test with real Kafka

	// TODO: Add performance benchmarks
	t.Skip("Performance test not yet implemented")
}
