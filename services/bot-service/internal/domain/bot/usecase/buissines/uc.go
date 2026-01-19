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
	producer                   deps.SubscriptionEventProducer
	repository                 deps.SubscriptionRepository
	deliveredMessageRepository deps.DeliveredMessageRepository
	sender                     deps.TelegramSender
	logger                     zerolog.Logger
}

// NewUseCase creates a new UseCase instance
// Note: sender is not passed here to break cyclic dependency
// Use SetSender after creating TelegramHandlers
func NewUseCase(
	producer deps.SubscriptionEventProducer,
	repository deps.SubscriptionRepository,
	deliveredMessageRepository deps.DeliveredMessageRepository,
	logger zerolog.Logger,
) *UseCase {
	return &UseCase{
		producer:                   producer,
		repository:                 repository,
		deliveredMessageRepository: deliveredMessageRepository,
		logger:                     logger,
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

// HandleToggleSubscription handles toggle subscription logic (Saga flow)
// Sends subscription.requested event - subscription-service determines action:
// - If not subscribed: creates pending subscription
// - If already subscribed: initiates unsubscription
// Actual result will come via Kafka confirmation/rejection events
func (uc *UseCase) HandleToggleSubscription(ctx context.Context, req *dto.ToggleSubscriptionRequest) error {
	uc.logger.Info().
		Int64("user_id", req.UserID).
		Str("channel_id", req.ChannelID).
		Msg("Processing toggle subscription request (Saga flow)")

	// Send subscription.requested event - subscription-service handles toggle logic
	subscription := &entities.Subscription{
		UserID:      req.UserID,
		ChannelID:   req.ChannelID,
		ChannelName: req.ChannelName,
		CreatedAt:   time.Now(),
	}
	if err := uc.producer.SendSubscriptionCreated(ctx, subscription); err != nil {
		uc.logger.Error().Err(err).Msg("Failed to send subscription request event")
		return fmt.Errorf("failed to send subscription request: %w", err)
	}

	return nil
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

// SendNews sends news to user via Telegram and saves delivery record
func (uc *UseCase) SendNews(ctx context.Context, news *entities.NewsMessage) error {
	if uc.sender == nil {
		uc.logger.Error().Msg("TelegramSender is not set")
		return boterrors.ErrMessageDeliveryFailed
	}

	uc.logger.Info().
		Uint("news_id", news.ID).
		Int64("user_id", news.UserID).
		Str("channel_id", news.ChannelID).
		Msg("Sending news to user")

	// Format message
	message := fmt.Sprintf("📰 <b>%s</b>\n\n%s", news.ChannelName, news.Content)

	// Send with or without media and get telegram message ID
	var telegramMsgID int
	var err error

	if len(news.MediaURLs) > 0 {
		telegramMsgID, err = uc.sender.SendMessageWithMediaAndGetID(ctx, news.UserID, message, news.MediaURLs)
	} else {
		telegramMsgID, err = uc.sender.SendMessageAndGetID(ctx, news.UserID, message)
	}

	if err != nil {
		uc.logger.Error().Err(err).
			Uint("news_id", news.ID).
			Int64("user_id", news.UserID).
			Msg("Failed to send news to user")
		return err
	}

	// Save delivery record for delete/edit sync
	if uc.deliveredMessageRepository != nil {
		deliveredMsg := &entities.DeliveredMessage{
			NewsID:            news.ID,
			UserID:            news.UserID,
			TelegramMessageID: telegramMsgID,
		}

		if saveErr := uc.deliveredMessageRepository.Save(ctx, deliveredMsg); saveErr != nil {
			uc.logger.Error().Err(saveErr).
				Uint("news_id", news.ID).
				Int64("user_id", news.UserID).
				Int("telegram_message_id", telegramMsgID).
				Msg("Failed to save delivered message record")
			// Don't fail the delivery, just log the error
		} else {
			uc.logger.Debug().
				Uint("news_id", news.ID).
				Int64("user_id", news.UserID).
				Int("telegram_message_id", telegramMsgID).
				Msg("Delivered message record saved")
		}
	}

	return nil
}

// DeleteNews deletes news messages from user chats
func (uc *UseCase) DeleteNews(ctx context.Context, newsID uint, userIDs []int64) error {
	if uc.sender == nil {
		uc.logger.Error().Msg("TelegramSender is not set")
		return boterrors.ErrMessageDeliveryFailed
	}

	uc.logger.Info().
		Uint("news_id", newsID).
		Int("users_count", len(userIDs)).
		Msg("Deleting news from user chats")

	var lastErr error
	successCount := 0

	for _, userID := range userIDs {
		// Get delivered message record
		deliveredMsg, err := uc.deliveredMessageRepository.GetByNewsIDAndUserID(ctx, newsID, userID)
		if err != nil {
			uc.logger.Warn().Err(err).
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Msg("Failed to get delivered message record, skipping")
			continue
		}

		// Delete message from Telegram
		if err := uc.sender.DeleteMessage(ctx, userID, deliveredMsg.TelegramMessageID); err != nil {
			uc.logger.Error().Err(err).
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Int("telegram_message_id", deliveredMsg.TelegramMessageID).
				Msg("Failed to delete message from Telegram")
			lastErr = err
			continue
		}

		// Delete delivery record
		if err := uc.deliveredMessageRepository.Delete(ctx, newsID, userID); err != nil {
			uc.logger.Warn().Err(err).
				Uint("news_id", newsID).
				Int64("user_id", userID).
				Msg("Failed to delete delivery record")
		}

		successCount++
		uc.logger.Debug().
			Uint("news_id", newsID).
			Int64("user_id", userID).
			Msg("News deleted from user chat")
	}

	uc.logger.Info().
		Uint("news_id", newsID).
		Int("success_count", successCount).
		Int("total_count", len(userIDs)).
		Msg("News delete operation completed")

	return lastErr
}

// EditNews edits news messages in user chats
func (uc *UseCase) EditNews(ctx context.Context, event *dto.NewsEditEvent) error {
	if uc.sender == nil {
		uc.logger.Error().Msg("TelegramSender is not set")
		return boterrors.ErrMessageDeliveryFailed
	}

	uc.logger.Info().
		Uint("news_id", event.NewsID).
		Int("users_count", len(event.UserIDs)).
		Str("channel_name", event.ChannelName).
		Msg("Editing news in user chats")

	// Format updated message
	message := fmt.Sprintf("📰 <b>%s</b>\n\n%s\n\n<i>(изменено)</i>", event.ChannelName, event.Content)

	var lastErr error
	successCount := 0

	for _, userID := range event.UserIDs {
		// Get delivered message record
		deliveredMsg, err := uc.deliveredMessageRepository.GetByNewsIDAndUserID(ctx, event.NewsID, userID)
		if err != nil {
			uc.logger.Warn().Err(err).
				Uint("news_id", event.NewsID).
				Int64("user_id", userID).
				Msg("Failed to get delivered message record, skipping")
			continue
		}

		// Edit message in Telegram
		if err := uc.sender.EditMessageText(ctx, userID, deliveredMsg.TelegramMessageID, message); err != nil {
			uc.logger.Error().Err(err).
				Uint("news_id", event.NewsID).
				Int64("user_id", userID).
				Int("telegram_message_id", deliveredMsg.TelegramMessageID).
				Msg("Failed to edit message in Telegram")
			lastErr = err
			continue
		}

		successCount++
		uc.logger.Debug().
			Uint("news_id", event.NewsID).
			Int64("user_id", userID).
			Msg("News edited in user chat")
	}

	uc.logger.Info().
		Uint("news_id", event.NewsID).
		Int("success_count", successCount).
		Int("total_count", len(event.UserIDs)).
		Msg("News edit operation completed")

	return lastErr
}
