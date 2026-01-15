// Package buissines contains business logic for the bot domain
package buissines

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/Conte777/newsflow/services/bot-service/internal/domain/bot/deps"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain/bot/dto"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain/bot/entities"
	boterrors "github.com/Conte777/newsflow/services/bot-service/internal/domain/bot/errors"
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

<b>Доступные команды:</b>
/subscribe @channel1 @channel2 - подписаться на каналы
/unsubscribe @channel1 - отписаться от канала
/list - список ваших подписок
/help - показать справку`

	return &dto.CommandResponse{Message: message}, nil
}

// HandleHelp handles /help command
func (uc *UseCase) HandleHelp(ctx context.Context) (*dto.CommandResponse, error) {
	message := `📚 <b>Справка по командам:</b>

/start - начать работу с ботом
/subscribe @channel1 @channel2 - подписаться на каналы
/unsubscribe @channel1 - отписаться от канала
/list - показать список ваших подписок
/help - показать эту справку

<b>Формат канала:</b> @channel_name (обязательно начинается с @)

<b>Примеры:</b>
• /subscribe @telegram @durov
• /unsubscribe @telegram
• /list`

	return &dto.CommandResponse{Message: message}, nil
}

// HandleSubscribe handles subscription request
func (uc *UseCase) HandleSubscribe(ctx context.Context, req *dto.SubscribeRequest) (*dto.CommandResponse, error) {
	if len(req.Channels) == 0 {
		return nil, boterrors.ErrNoChannelsSpecified
	}

	uc.logger.Info().
		Int64("user_id", req.UserID).
		Strs("channels", req.Channels).
		Msg("Processing subscription request")

	var subscribed []string
	var errors []string

	for _, channel := range req.Channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}

		// Validate channel format
		if !strings.HasPrefix(channel, "@") {
			errors = append(errors, fmt.Sprintf("%s - неверный формат (нужен @)", channel))
			continue
		}

		// Create subscription entity
		subscription := &entities.Subscription{
			UserID:      req.UserID,
			ChannelID:   channel,
			ChannelName: strings.TrimPrefix(channel, "@"),
			CreatedAt:   time.Now(),
		}

		// Send event to Kafka
		if err := uc.producer.SendSubscriptionCreated(ctx, subscription); err != nil {
			uc.logger.Error().Err(err).Str("channel", channel).Msg("Failed to send subscription event")
			errors = append(errors, fmt.Sprintf("%s - ошибка отправки", channel))
			continue
		}

		subscribed = append(subscribed, channel)
	}

	var message string
	if len(subscribed) > 0 {
		message = fmt.Sprintf("✅ Подписка оформлена на: %s", strings.Join(subscribed, ", "))
	}
	if len(errors) > 0 {
		if message != "" {
			message += "\n\n"
		}
		message += fmt.Sprintf("❌ Ошибки:\n%s", strings.Join(errors, "\n"))
	}
	if message == "" {
		message = "❌ Не указаны каналы для подписки"
	}

	return &dto.CommandResponse{Message: message}, nil
}

// HandleUnsubscribe handles unsubscription request
func (uc *UseCase) HandleUnsubscribe(ctx context.Context, req *dto.UnsubscribeRequest) (*dto.CommandResponse, error) {
	if len(req.Channels) == 0 {
		return nil, boterrors.ErrNoChannelsSpecified
	}

	uc.logger.Info().
		Int64("user_id", req.UserID).
		Strs("channels", req.Channels).
		Msg("Processing unsubscription request")

	var unsubscribed []string
	var errors []string

	for _, channel := range req.Channels {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			continue
		}

		// Validate channel format
		if !strings.HasPrefix(channel, "@") {
			errors = append(errors, fmt.Sprintf("%s - неверный формат", channel))
			continue
		}

		// Send event to Kafka
		if err := uc.producer.SendSubscriptionDeleted(ctx, req.UserID, channel); err != nil {
			uc.logger.Error().Err(err).Str("channel", channel).Msg("Failed to send unsubscription event")
			errors = append(errors, fmt.Sprintf("%s - ошибка отправки", channel))
			continue
		}

		unsubscribed = append(unsubscribed, channel)
	}

	var message string
	if len(unsubscribed) > 0 {
		message = fmt.Sprintf("✅ Отписка выполнена от: %s", strings.Join(unsubscribed, ", "))
	}
	if len(errors) > 0 {
		if message != "" {
			message += "\n\n"
		}
		message += fmt.Sprintf("❌ Ошибки:\n%s", strings.Join(errors, "\n"))
	}
	if message == "" {
		message = "❌ Не указаны каналы для отписки"
	}

	return &dto.CommandResponse{Message: message}, nil
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
