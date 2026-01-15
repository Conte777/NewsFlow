// Package buissines contains business logic for the bot domain
package buissines

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/deps"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/dto"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
	boterrors "github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/errors"
)

// UseCase contains business logic for bot operations
type UseCase struct {
	producer   deps.SubscriptionEventProducer
	repository deps.SubscriptionRepository
	sender     deps.TelegramSender
	logger     zerolog.Logger
}

// NewUseCase creates a new UseCase instance
// Note: sender is not passed here to break cyclic dependency
// Use SetSender after creating TelegramHandlers
func NewUseCase(producer deps.SubscriptionEventProducer, repository deps.SubscriptionRepository, logger zerolog.Logger) *UseCase {
	return &UseCase{
		producer:   producer,
		repository: repository,
		logger:     logger,
	}
}

// SetSender sets the TelegramSender after construction
// This is called by fx.Invoke to resolve cyclic dependency
func (uc *UseCase) SetSender(sender deps.TelegramSender) {
	uc.sender = sender
}

// HandleStart handles /start command
func (uc *UseCase) HandleStart(ctx context.Context, req *dto.StartCommandRequest) (*dto.CommandResponse, error) {
	uc.logger.Info().
		Int64("user_id", req.UserID).
		Str("username", req.Username).
		Msg("User started bot")

	message := `👋 <b>Добро пожаловать в NewsFlow Bot!</b>

Я помогу вам получать новости из ваших любимых Telegram-каналов.

<b>Как подписаться/отписаться:</b>
Перешлите мне сообщение из публичного канала — я автоматически подпишу вас или отпишу, если вы уже подписаны.

<b>Доступные команды:</b>
/list - список ваших подписок
/help - показать справку`

	return &dto.CommandResponse{Message: message}, nil
}

// HandleHelp handles /help command
func (uc *UseCase) HandleHelp(ctx context.Context) (*dto.CommandResponse, error) {
	message := `📚 <b>Справка:</b>

<b>Подписка на канал:</b>
Перешлите мне сообщение из публичного канала — я подпишу вас на него.

<b>Отписка от канала:</b>
Перешлите мне сообщение из канала, на который вы уже подписаны — я отпишу вас от него.

<b>Команды:</b>
/start - начать работу с ботом
/list - показать список ваших подписок
/help - показать эту справку`

	return &dto.CommandResponse{Message: message}, nil
}

// HandleToggleSubscription handles toggle subscription logic
// If user is subscribed - unsubscribe, if not - subscribe
func (uc *UseCase) HandleToggleSubscription(ctx context.Context, req *dto.ToggleSubscriptionRequest) (*dto.ToggleSubscriptionResponse, error) {
	uc.logger.Info().
		Int64("user_id", req.UserID).
		Str("channel_id", req.ChannelID).
		Msg("Processing toggle subscription request")

	// Check if already subscribed via gRPC
	isSubscribed, err := uc.repository.CheckSubscription(ctx, req.UserID, req.ChannelID)
	if err != nil {
		uc.logger.Error().Err(err).Msg("Failed to check subscription status")
		return nil, fmt.Errorf("failed to check subscription: %w", err)
	}

	if isSubscribed {
		// Unsubscribe
		if err := uc.producer.SendSubscriptionDeleted(ctx, req.UserID, req.ChannelID); err != nil {
			uc.logger.Error().Err(err).Msg("Failed to send unsubscription event")
			return nil, fmt.Errorf("failed to unsubscribe: %w", err)
		}
		return &dto.ToggleSubscriptionResponse{
			Message: fmt.Sprintf("✅ Вы отписались от канала %s", req.ChannelID),
			Action:  "unsubscribed",
		}, nil
	}

	// Subscribe
	subscription := &entities.Subscription{
		UserID:      req.UserID,
		ChannelID:   req.ChannelID,
		ChannelName: req.ChannelName,
		CreatedAt:   time.Now(),
	}
	if err := uc.producer.SendSubscriptionCreated(ctx, subscription); err != nil {
		uc.logger.Error().Err(err).Msg("Failed to send subscription event")
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return &dto.ToggleSubscriptionResponse{
		Message: fmt.Sprintf("✅ Вы подписались на канал %s", req.ChannelID),
		Action:  "subscribed",
	}, nil
}

// HandleListSubscriptions handles listing user subscriptions
func (uc *UseCase) HandleListSubscriptions(ctx context.Context, userID int64) (*dto.SubscriptionListResponse, error) {
	uc.logger.Info().
		Int64("user_id", userID).
		Msg("Listing user subscriptions")

	subs, err := uc.repository.GetUserSubscriptions(ctx, userID)
	if err != nil {
		uc.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to get subscriptions")
		return nil, err
	}

	items := make([]dto.SubscriptionItem, len(subs))
	for i, sub := range subs {
		items[i] = dto.SubscriptionItem{
			ChannelID:   sub.ChannelID,
			ChannelName: sub.ChannelName,
			CreatedAt:   sub.CreatedAt,
		}
	}

	return &dto.SubscriptionListResponse{Subscriptions: items}, nil
}

// SendNews sends news to user via Telegram
func (uc *UseCase) SendNews(ctx context.Context, news *entities.NewsMessage) error {
	if uc.sender == nil {
		uc.logger.Error().Msg("TelegramSender is not set")
		return boterrors.ErrMessageDeliveryFailed
	}

	uc.logger.Info().
		Str("news_id", news.ID).
		Int64("user_id", news.UserID).
		Str("channel_id", news.ChannelID).
		Msg("Sending news to user")

	// Format message
	message := fmt.Sprintf("📰 <b>%s</b>\n\n%s", news.ChannelName, news.Content)

	// Send with or without media
	if len(news.MediaURLs) > 0 {
		return uc.sender.SendMessageWithMedia(ctx, news.UserID, message, news.MediaURLs)
	}

	return uc.sender.SendMessage(ctx, news.UserID, message)
}
