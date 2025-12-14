package kafka_test

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/kafka"
)

// ExampleKafkaProducer demonstrates how to use KafkaProducer
// to send news items to Kafka topic
func ExampleKafkaProducer() {
	// Create logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// Configure producer
	config := kafka.ProducerConfig{
		Brokers:         []string{"localhost:9092", "localhost:9093"},
		Topic:           "news.received",
		Logger:          logger,
		MaxMessageBytes: 1000000, // 1MB
		MaxRetries:      5,
		ErrorCallback: func(news *domain.NewsItem, err error) {
			// Handle send errors (e.g., send to DLQ, retry, alert)
			logger.Error().
				Err(err).
				Str("channel_id", news.ChannelID).
				Int("message_id", news.MessageID).
				Msg("Failed to send news to Kafka")
		},
	}

	// Create producer
	producer, err := kafka.NewKafkaProducer(config)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Kafka producer")
	}
	defer producer.Close()

	// Create news item
	news := &domain.NewsItem{
		ChannelID:   "tech_news_channel",
		ChannelName: "Tech News Daily",
		MessageID:   123456,
		Content:     "Breaking: New AI model achieves human-level performance",
		MediaURLs: []string{
			"https://cdn.example.com/ai-news.jpg",
			"https://cdn.example.com/ai-demo.mp4",
		},
		Date: time.Now(),
	}

	// Send news to Kafka
	ctx := context.Background()
	if err := producer.SendNewsReceived(ctx, news); err != nil {
		logger.Error().Err(err).Msg("Failed to send news item")
		return
	}

	logger.Info().
		Str("channel_id", news.ChannelID).
		Int("message_id", news.MessageID).
		Msg("News item sent to Kafka successfully")
}

// ExampleKafkaProducer_bulkSend demonstrates sending multiple news items
func ExampleKafkaProducer_bulkSend() {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	config := kafka.ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "news.received",
		Logger:  logger,
	}

	producer, err := kafka.NewKafkaProducer(config)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create producer")
	}
	defer producer.Close()

	ctx := context.Background()

	// Send 10 news items
	for i := 1; i <= 10; i++ {
		news := &domain.NewsItem{
			ChannelID:   "tech_channel",
			ChannelName: "Technology Updates",
			MessageID:   i,
			Content:     fmt.Sprintf("News item #%d: Latest tech updates", i),
			MediaURLs:   []string{fmt.Sprintf("https://example.com/img%d.jpg", i)},
			Date:        time.Now(),
		}

		if err := producer.SendNewsReceived(ctx, news); err != nil {
			logger.Error().Err(err).Int("message_id", i).Msg("Failed to send")
		} else {
			logger.Debug().Int("message_id", i).Msg("Sent successfully")
		}
	}

	logger.Info().Msg("Bulk send completed")
}

// ExampleKafkaProducer_withContext demonstrates using context for cancellation
func ExampleKafkaProducer_withContext() {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	config := kafka.ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "news.received",
		Logger:  logger,
	}

	producer, err := kafka.NewKafkaProducer(config)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create producer")
	}
	defer producer.Close()

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	news := &domain.NewsItem{
		ChannelID:   "breaking_news",
		ChannelName: "Breaking News",
		MessageID:   999,
		Content:     "Urgent: Important announcement",
		Date:        time.Now(),
	}

	if err := producer.SendNewsReceived(ctx, news); err != nil {
		logger.Error().Err(err).Msg("Send failed or timed out")
	} else {
		logger.Info().Msg("News sent successfully")
	}
}
