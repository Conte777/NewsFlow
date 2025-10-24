package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yourusername/telegram-news-feed/news-service/internal/domain"
	"github.com/rs/zerolog"
)

// newsUseCase implements domain.NewsUseCase
type newsUseCase struct {
	newsRepo          domain.NewsRepository
	deliveredRepo     domain.DeliveredNewsRepository
	kafkaProducer     domain.KafkaProducer
	subscriptionSvc   domain.SubscriptionService
	logger            zerolog.Logger
}

// NewNewsUseCase creates a new news use case
func NewNewsUseCase(
	newsRepo domain.NewsRepository,
	deliveredRepo domain.DeliveredNewsRepository,
	kafkaProducer domain.KafkaProducer,
	subscriptionSvc domain.SubscriptionService,
	logger zerolog.Logger,
) domain.NewsUseCase {
	return &newsUseCase{
		newsRepo:        newsRepo,
		deliveredRepo:   deliveredRepo,
		kafkaProducer:   kafkaProducer,
		subscriptionSvc: subscriptionSvc,
		logger:          logger,
	}
}

// ProcessNewsReceived processes a news received event
func (u *newsUseCase) ProcessNewsReceived(ctx context.Context, channelID, channelName string, messageID int, content string, mediaURLs []string) error {
	// Check if news already exists
	exists, err := u.newsRepo.Exists(ctx, channelID, messageID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Msg("Failed to check news existence")
		return err
	}

	if exists {
		u.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Msg("News already exists, skipping")
		return nil
	}

	// Serialize media URLs to JSON
	mediaURLsJSON, err := json.Marshal(mediaURLs)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to marshal media URLs")
		return err
	}

	// Create news record
	news := &domain.News{
		ChannelID:   channelID,
		ChannelName: channelName,
		MessageID:   messageID,
		Content:     content,
		MediaURLs:   string(mediaURLsJSON),
	}

	if err := u.newsRepo.Create(ctx, news); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Msg("Failed to create news")
		return err
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Int("message_id", messageID).
		Uint("news_id", news.ID).
		Msg("News created successfully")

	// Get channel subscribers
	subscribers, err := u.subscriptionSvc.GetChannelSubscribers(ctx, channelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to get channel subscribers")
		return err
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Int("subscribers_count", len(subscribers)).
		Msg("Delivering news to subscribers")

	// Deliver news to all subscribers
	return u.DeliverNewsToUsers(ctx, news.ID, subscribers)
}

// DeliverNewsToUsers delivers news to subscribed users
func (u *newsUseCase) DeliverNewsToUsers(ctx context.Context, newsID uint, userIDs []int64) error {
	// Get news details
	news, err := u.newsRepo.GetByID(ctx, newsID)
	if err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", newsID).
			Msg("Failed to get news")
		return err
	}

	// Parse media URLs
	var mediaURLs []string
	if news.MediaURLs != "" {
		if err := json.Unmarshal([]byte(news.MediaURLs), &mediaURLs); err != nil {
			u.logger.Error().Err(err).Msg("Failed to unmarshal media URLs")
			mediaURLs = []string{}
		}
	}

	// Send delivery event for each user
	for _, userID := range userIDs {
		// Check if already delivered
		delivered, err := u.deliveredRepo.IsDelivered(ctx, newsID, userID)
		if err != nil {
			u.logger.Error().Err(err).
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Msg("Failed to check delivery status")
			continue
		}

		if delivered {
			u.logger.Debug().
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Msg("News already delivered to user")
			continue
		}

		// Send news delivery event
		if err := u.kafkaProducer.SendNewsDelivery(ctx, newsID, userID, news.ChannelID, news.ChannelName, news.Content, mediaURLs); err != nil {
			u.logger.Error().Err(err).
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Msg("Failed to send news delivery event")
			continue
		}

		u.logger.Debug().
			Uint("news_id", newsID).
			Int64("user_id", userID).
			Msg("News delivery event sent")
	}

	return nil
}

// MarkAsDelivered marks news as delivered to user
func (u *newsUseCase) MarkAsDelivered(ctx context.Context, newsID uint, userID int64) error {
	delivered := &domain.DeliveredNews{
		NewsID: newsID,
		UserID: userID,
	}

	if err := u.deliveredRepo.Create(ctx, delivered); err != nil {
		if err == domain.ErrAlreadyDelivered {
			u.logger.Debug().
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Msg("News already marked as delivered")
			return nil
		}

		u.logger.Error().Err(err).
			Uint("news_id", newsID).
			Int64("user_id", userID).
			Msg("Failed to mark news as delivered")
		return err
	}

	u.logger.Info().
		Uint("news_id", newsID).
		Int64("user_id", userID).
		Msg("News marked as delivered")

	return nil
}

// GetUserDeliveredNews retrieves user's delivered news history
func (u *newsUseCase) GetUserDeliveredNews(ctx context.Context, userID int64, limit int) ([]domain.DeliveredNews, error) {
	if userID <= 0 {
		return nil, domain.ErrInvalidUserID
	}

	deliveredNews, err := u.deliveredRepo.GetUserDeliveredNews(ctx, userID, limit)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Msg("Failed to get user delivered news")
		return nil, fmt.Errorf("failed to get user delivered news: %w", err)
	}

	return deliveredNews, nil
}
