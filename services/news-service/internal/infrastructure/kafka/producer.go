package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"github.com/Conte777/NewsFlow/services/news-service/config"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/deps"
)

const (
	topicNewsDelivery = "news.deliver"
	topicNewsDelete   = "news.delete"
	topicNewsEdit     = "news.edit"
)

type NewsDeliveryMessage struct {
	NewsID      uint     `json:"news_id"`
	UserIDs     []int64  `json:"user_ids"`
	ChannelID   string   `json:"channel_id"`
	ChannelName string   `json:"channel_name"`
	MessageID   int      `json:"message_id"` // Telegram message ID for copyMessage
	Content     string   `json:"content"`
	MediaURLs   []string `json:"media_urls"`
	Timestamp   int64    `json:"timestamp"`
}

type NewsDeleteMessage struct {
	NewsID  uint    `json:"news_id"`
	UserIDs []int64 `json:"user_ids"`
}

type NewsEditMessage struct {
	NewsID      uint     `json:"news_id"`
	UserIDs     []int64  `json:"user_ids"`
	Content     string   `json:"content"`
	ChannelName string   `json:"channel_name"`
	MediaURLs   []string `json:"media_urls"`
}

type Producer struct {
	writer *kafka.Writer
	logger zerolog.Logger
}

func NewProducer(cfg *config.KafkaConfig, logger zerolog.Logger) (deps.KafkaProducer, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}

	logger.Info().
		Strs("brokers", cfg.Brokers).
		Msg("Kafka producer initialized")

	return &Producer{
		writer: writer,
		logger: logger,
	}, nil
}

func (p *Producer) SendNewsDelivery(ctx context.Context, newsID uint, userIDs []int64, channelID, channelName string, messageID int, content string, mediaURLs []string) error {
	msg := NewsDeliveryMessage{
		NewsID:      newsID,
		UserIDs:     userIDs,
		ChannelID:   channelID,
		ChannelName: channelName,
		MessageID:   messageID,
		Content:     content,
		MediaURLs:   mediaURLs,
		Timestamp:   time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("news-%d", newsID)

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topicNewsDelivery,
		Key:   []byte(key),
		Value: data,
	})
	if err != nil {
		p.logger.Error().Err(err).
			Uint("news_id", newsID).
			Int("users_count", len(userIDs)).
			Msg("Failed to send batch news delivery message")
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Debug().
		Uint("news_id", newsID).
		Int("users_count", len(userIDs)).
		Msg("Batch news delivery message sent")

	return nil
}

func (p *Producer) SendNewsDelete(ctx context.Context, newsID uint, userIDs []int64) error {
	msg := NewsDeleteMessage{
		NewsID:  newsID,
		UserIDs: userIDs,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("news-%d", newsID)

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topicNewsDelete,
		Key:   []byte(key),
		Value: data,
	})
	if err != nil {
		p.logger.Error().Err(err).
			Uint("news_id", newsID).
			Int("users_count", len(userIDs)).
			Msg("Failed to send news delete message")
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Debug().
		Uint("news_id", newsID).
		Int("users_count", len(userIDs)).
		Msg("News delete message sent")

	return nil
}

func (p *Producer) SendNewsEdit(ctx context.Context, newsID uint, userIDs []int64, content, channelName string, mediaURLs []string) error {
	msg := NewsEditMessage{
		NewsID:      newsID,
		UserIDs:     userIDs,
		Content:     content,
		ChannelName: channelName,
		MediaURLs:   mediaURLs,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("news-%d", newsID)

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topicNewsEdit,
		Key:   []byte(key),
		Value: data,
	})
	if err != nil {
		p.logger.Error().Err(err).
			Uint("news_id", newsID).
			Int("users_count", len(userIDs)).
			Msg("Failed to send news edit message")
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Debug().
		Uint("news_id", newsID).
		Int("users_count", len(userIDs)).
		Msg("News edit message sent")

	return nil
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
