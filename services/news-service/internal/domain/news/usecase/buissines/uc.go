package buissines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/deps"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/dto"
	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/entities"
	domainerrors "github.com/Conte777/NewsFlow/services/news-service/internal/domain/news/errors"
	pkgerrors "github.com/Conte777/NewsFlow/services/news-service/pkg/errors"
	"github.com/rs/zerolog"
)

// UseCase implements news business logic
type UseCase struct {
	newsRepo        deps.NewsRepository
	deliveredRepo   deps.DeliveredNewsRepository
	kafkaProducer   deps.KafkaProducer
	subscriptionSvc deps.SubscriptionClient
	logger          zerolog.Logger
}

// NewUseCase creates a new news use case
func NewUseCase(
	newsRepo deps.NewsRepository,
	deliveredRepo deps.DeliveredNewsRepository,
	kafkaProducer deps.KafkaProducer,
	subscriptionSvc deps.SubscriptionClient,
	logger zerolog.Logger,
) *UseCase {
	return &UseCase{
		newsRepo:        newsRepo,
		deliveredRepo:   deliveredRepo,
		kafkaProducer:   kafkaProducer,
		subscriptionSvc: subscriptionSvc,
		logger:          logger,
	}
}

// ProcessNewsReceived processes a news received event
func (u *UseCase) ProcessNewsReceived(ctx context.Context, req *dto.ProcessNewsRequest) (*dto.ProcessNewsResponse, error) {
	if req.ChannelID == "" {
		return nil, domainerrors.ErrInvalidChannelID
	}

	exists, err := u.newsRepo.Exists(ctx, req.ChannelID, req.MessageID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", req.ChannelID).
			Int("message_id", req.MessageID).
			Msg("Failed to check news existence")
		return nil, err
	}

	if exists {
		u.logger.Debug().
			Str("channel_id", req.ChannelID).
			Int("message_id", req.MessageID).
			Msg("News already exists, skipping")
		return nil, pkgerrors.NewConflictError("news already exists")
	}

	mediaURLsJSON, err := json.Marshal(req.MediaURLs)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to marshal media URLs")
		return nil, err
	}

	news := &entities.News{
		ChannelID:   req.ChannelID,
		ChannelName: req.ChannelName,
		MessageID:   req.MessageID,
		Content:     req.Content,
		MediaURLs:   string(mediaURLsJSON),
	}

	if err := u.newsRepo.Create(ctx, news); err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", req.ChannelID).
			Int("message_id", req.MessageID).
			Msg("Failed to create news")
		return nil, err
	}

	u.logger.Info().
		Str("channel_id", req.ChannelID).
		Int("message_id", req.MessageID).
		Uint("news_id", news.ID).
		Msg("News created successfully")

	subscribers, err := u.getChannelSubscribersWithFallback(ctx, req.ChannelID)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", req.ChannelID).
			Msg("Failed to get channel subscribers")
		return nil, err
	}

	u.logger.Info().
		Str("channel_id", req.ChannelID).
		Int("subscribers_count", len(subscribers)).
		Msg("Delivering news to subscribers")

	if err := u.DeliverNewsToUsers(ctx, news.ID, subscribers); err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", news.ID).
			Msg("Failed to deliver news to users")
	}

	return &dto.ProcessNewsResponse{NewsID: news.ID}, nil
}

// DeliverNewsToUsers delivers news to subscribed users (batch format)
func (u *UseCase) DeliverNewsToUsers(ctx context.Context, newsID uint, userIDs []int64) error {
	news, err := u.newsRepo.GetByID(ctx, newsID)
	if err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", newsID).
			Msg("Failed to get news")
		return err
	}

	var mediaURLs []string
	if news.MediaURLs != "" {
		if err := json.Unmarshal([]byte(news.MediaURLs), &mediaURLs); err != nil {
			u.logger.Error().Err(err).Msg("Failed to unmarshal media URLs")
			mediaURLs = []string{}
		}
	}

	pendingUserIDs := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
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

		pendingUserIDs = append(pendingUserIDs, userID)
	}

	if len(pendingUserIDs) == 0 {
		u.logger.Debug().
			Uint("news_id", newsID).
			Msg("No pending users for news delivery")
		return nil
	}

	if err := u.kafkaProducer.SendNewsDelivery(ctx, newsID, pendingUserIDs, news.ChannelID, news.ChannelName, news.MessageID, news.Content, mediaURLs); err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", newsID).
			Int("pending_users_count", len(pendingUserIDs)).
			Msg("Failed to send batch news delivery event")
		return err
	}

	u.logger.Info().
		Uint("news_id", newsID).
		Int("pending_users_count", len(pendingUserIDs)).
		Msg("Batch news delivery event sent")

	return nil
}

// MarkAsDelivered marks news as delivered to user
func (u *UseCase) MarkAsDelivered(ctx context.Context, newsID uint, userID int64) error {
	delivered := &entities.DeliveredNews{
		NewsID: newsID,
		UserID: userID,
	}

	if err := u.deliveredRepo.Create(ctx, delivered); err != nil {
		if errors.Is(err, domainerrors.ErrAlreadyDelivered) {
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
func (u *UseCase) GetUserDeliveredNews(ctx context.Context, req *dto.GetUserNewsRequest) (*dto.GetUserNewsResponse, error) {
	if req.UserID <= 0 {
		return nil, domainerrors.ErrInvalidUserID
	}

	deliveredNews, err := u.deliveredRepo.GetUserDeliveredNews(ctx, req.UserID, req.Limit)
	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", req.UserID).
			Msg("Failed to get user delivered news")
		return nil, fmt.Errorf("failed to get user delivered news: %w", err)
	}

	newsItems := make([]dto.NewsItem, 0, len(deliveredNews))
	for _, dn := range deliveredNews {
		var mediaURLs []string
		if dn.News.MediaURLs != "" {
			if err := json.Unmarshal([]byte(dn.News.MediaURLs), &mediaURLs); err != nil {
				mediaURLs = []string{}
			}
		}

		newsItems = append(newsItems, dto.NewsItem{
			ID:          dn.News.ID,
			ChannelID:   dn.News.ChannelID,
			ChannelName: dn.News.ChannelName,
			Content:     dn.News.Content,
			MediaURLs:   mediaURLs,
			DeliveredAt: dn.DeliveredAt,
		})
	}

	return &dto.GetUserNewsResponse{News: newsItems}, nil
}

// getChannelSubscribersWithFallback tries gRPC first, falls back to delivered_news
func (u *UseCase) getChannelSubscribersWithFallback(ctx context.Context, channelID string) ([]int64, error) {
	subscribers, err := u.subscriptionSvc.GetChannelSubscribers(ctx, channelID)
	if err == nil {
		return subscribers, nil
	}

	u.logger.Warn().Err(err).
		Str("channel_id", channelID).
		Msg("Subscription service unavailable, using fallback from delivered_news")

	fallbackUsers, fallbackErr := u.deliveredRepo.GetUsersByChannelID(ctx, channelID)
	if fallbackErr != nil {
		u.logger.Error().Err(fallbackErr).
			Str("channel_id", channelID).
			Msg("Fallback to delivered_news also failed")
		return nil, fmt.Errorf("subscription service unavailable and fallback failed: %w", err)
	}

	u.logger.Info().
		Str("channel_id", channelID).
		Int("fallback_users_count", len(fallbackUsers)).
		Msg("Using fallback subscribers from delivered_news")

	return fallbackUsers, nil
}

// ProcessNewsDeleted processes news deleted events
func (u *UseCase) ProcessNewsDeleted(ctx context.Context, event *dto.NewsDeletedEvent) error {
	if event.ChannelID == "" {
		return domainerrors.ErrInvalidChannelID
	}

	if len(event.MessageIDs) == 0 {
		u.logger.Debug().Str("channel_id", event.ChannelID).Msg("No message IDs to delete")
		return nil
	}

	deletedAt := time.Now()
	if event.DeletedAt > 0 {
		deletedAt = time.Unix(event.DeletedAt, 0)
	}

	// Soft delete news and get their IDs
	deletedIDs, err := u.newsRepo.SoftDeleteBatch(ctx, event.ChannelID, event.MessageIDs)
	if err != nil {
		u.logger.Error().Err(err).
			Str("channel_id", event.ChannelID).
			Ints("message_ids", event.MessageIDs).
			Msg("Failed to soft delete news")
		return err
	}

	if len(deletedIDs) == 0 {
		u.logger.Debug().
			Str("channel_id", event.ChannelID).
			Ints("message_ids", event.MessageIDs).
			Msg("No news found to delete")
		return nil
	}

	u.logger.Info().
		Str("channel_id", event.ChannelID).
		Int("deleted_count", len(deletedIDs)).
		Msg("News soft deleted")

	// For each deleted news, notify users who received it
	for _, newsID := range deletedIDs {
		userIDs, err := u.deliveredRepo.GetUsersByNewsID(ctx, newsID)
		if err != nil {
			u.logger.Error().Err(err).
				Uint("news_id", newsID).
				Msg("Failed to get users for deleted news")
			// Continue to fallback instead of skipping
		}

		// Fallback: if no delivered users found, try to get channel subscribers
		if len(userIDs) == 0 {
			u.logger.Debug().Uint("news_id", newsID).Str("channel_id", event.ChannelID).
				Msg("No delivered users found, falling back to channel subscribers")

			fallbackUsers, fallbackErr := u.subscriptionSvc.GetChannelSubscribers(ctx, event.ChannelID)
			if fallbackErr != nil {
				u.logger.Warn().Err(fallbackErr).Str("channel_id", event.ChannelID).
					Msg("Failed to get channel subscribers for fallback")
				continue
			}
			userIDs = fallbackUsers
		}

		if len(userIDs) == 0 {
			u.logger.Debug().Uint("news_id", newsID).Msg("No users to notify about deleted news")
			continue
		}

		if err := u.kafkaProducer.SendNewsDelete(ctx, newsID, userIDs); err != nil {
			u.logger.Error().Err(err).
				Uint("news_id", newsID).
				Int("users_count", len(userIDs)).
				Msg("Failed to send news delete event")
			continue
		}

		u.logger.Info().
			Uint("news_id", newsID).
			Int("users_count", len(userIDs)).
			Msg("News delete event sent to users")

		if err := u.deliveredRepo.SoftDeleteByNewsIDs(ctx, []uint{newsID}, deletedAt); err != nil {
			u.logger.Warn().Err(err).
				Uint("news_id", newsID).
				Msg("Failed to mark delivered news as deleted")
		}
	}

	return nil
}

// ProcessNewsEdited processes news edited events
func (u *UseCase) ProcessNewsEdited(ctx context.Context, event *dto.NewsEditedEvent) error {
	if event.ChannelID == "" {
		return domainerrors.ErrInvalidChannelID
	}

	// Find existing news
	news, err := u.newsRepo.GetByChannelAndMessageID(ctx, event.ChannelID, event.MessageID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrNewsNotFound) {
			u.logger.Debug().
				Str("channel_id", event.ChannelID).
				Int("message_id", event.MessageID).
				Msg("Edited news not found in DB, skipping")
			return nil
		}
		u.logger.Error().Err(err).
			Str("channel_id", event.ChannelID).
			Int("message_id", event.MessageID).
			Msg("Failed to find news for edit")
		return err
	}

	// Update news content
	mediaURLsJSON, err := json.Marshal(event.MediaURLs)
	if err != nil {
		u.logger.Error().Err(err).Msg("Failed to marshal media URLs")
		mediaURLsJSON = []byte("[]")
	}

	news.Content = event.Content
	news.MediaURLs = string(mediaURLsJSON)
	if event.ChannelName != "" {
		news.ChannelName = event.ChannelName
	}

	if err := u.newsRepo.Update(ctx, news); err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", news.ID).
			Msg("Failed to update news")
		return err
	}

	u.logger.Info().
		Uint("news_id", news.ID).
		Str("channel_id", event.ChannelID).
		Int("message_id", event.MessageID).
		Msg("News updated")

	updatedAt := time.Now()
	if event.EditedAt > 0 {
		updatedAt = time.Unix(event.EditedAt, 0)
	}

	if err := u.deliveredRepo.UpdateTimestampByNewsID(ctx, news.ID, updatedAt); err != nil {
		u.logger.Warn().Err(err).
			Uint("news_id", news.ID).
			Msg("Failed to update delivered news timestamp")
	}

	// Get users who received this news
	userIDs, err := u.deliveredRepo.GetUsersByNewsID(ctx, news.ID)
	if err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", news.ID).
			Msg("Failed to get users for edited news")
		return err
	}

	if len(userIDs) == 0 {
		u.logger.Debug().
			Uint("news_id", news.ID).
			Msg("No users to notify about edited news")
		return nil
	}

	// Send edit event to bot-service
	if err := u.kafkaProducer.SendNewsEdit(ctx, news.ID, userIDs, event.Content, news.ChannelName, event.MediaURLs); err != nil {
		u.logger.Error().Err(err).
			Uint("news_id", news.ID).
			Int("users_count", len(userIDs)).
			Msg("Failed to send news edit event")
		return err
	}

	u.logger.Info().
		Uint("news_id", news.ID).
		Int("users_count", len(userIDs)).
		Msg("News edit event sent to users")

	return nil
}
