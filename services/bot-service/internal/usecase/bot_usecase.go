package usecase

import (
	"context"
	"fmt"

	"github.com/yourusername/telegram-news-feed/bot-service/internal/domain"
	"github.com/rs/zerolog"
)

// botUseCase implements domain.BotUseCase
type botUseCase struct {
	kafkaProducer domain.KafkaProducer
	telegramBot   domain.TelegramBot
	logger        zerolog.Logger
}

// NewBotUseCase creates a new bot use case
func NewBotUseCase(
	kafkaProducer domain.KafkaProducer,
	telegramBot domain.TelegramBot,
	logger zerolog.Logger,
) domain.BotUseCase {
	return &botUseCase{
		kafkaProducer: kafkaProducer,
		telegramBot:   telegramBot,
		logger:        logger,
	}
}

// HandleStart handles /start command
func (u *botUseCase) HandleStart(ctx context.Context, userID int64, username string) (string, error) {
	u.logger.Info().
		Int64("user_id", userID).
		Str("username", username).
		Msg("User started bot")

	message := fmt.Sprintf(
		"Добро пожаловать, %s! 👋\n\n"+
			"Я помогу вам получать новости из ваших любимых Telegram каналов.\n\n"+
			"Используйте:\n"+
			"/subscribe <channel_id> - подписаться на канал\n"+
			"/unsubscribe <channel_id> - отписаться от канала\n"+
			"/list - список ваших подписок\n"+
			"/help - помощь",
		username,
	)

	return message, nil
}

// HandleHelp handles /help command
func (u *botUseCase) HandleHelp(ctx context.Context) (string, error) {
	message := "📚 Доступные команды:\n\n" +
		"/start - начать работу с ботом\n" +
		"/subscribe <channel_id> - подписаться на канал\n" +
		"/unsubscribe <channel_id> - отписаться от канала\n" +
		"/list - показать список подписок\n" +
		"/help - показать это сообщение\n\n" +
		"Пример: /subscribe @channelname"

	return message, nil
}

// HandleSubscribe handles subscription request
func (u *botUseCase) HandleSubscribe(ctx context.Context, userID int64, channelID string) (string, error) {
	if channelID == "" {
		return "", domain.ErrInvalidChannelID
	}

	subscription := &domain.Subscription{
		UserID:      userID,
		ChannelID:   channelID,
		ChannelName: channelID,
	}

	// Send subscription event to Kafka
	if err := u.kafkaProducer.SendSubscriptionCreated(ctx, subscription); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to send subscription created event")
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Subscription created")

	return fmt.Sprintf("✅ Вы успешно подписались на канал %s", channelID), nil
}

// HandleUnsubscribe handles unsubscription request
func (u *botUseCase) HandleUnsubscribe(ctx context.Context, userID int64, channelID string) (string, error) {
	if channelID == "" {
		return "", domain.ErrInvalidChannelID
	}

	// Send unsubscription event to Kafka
	if err := u.kafkaProducer.SendSubscriptionDeleted(ctx, userID, channelID); err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("channel_id", channelID).
			Msg("Failed to send subscription deleted event")
		return "", fmt.Errorf("failed to delete subscription: %w", err)
	}

	u.logger.Info().
		Int64("user_id", userID).
		Str("channel_id", channelID).
		Msg("Subscription deleted")

	return fmt.Sprintf("✅ Вы отписались от канала %s", channelID), nil
}

// HandleListSubscriptions handles listing user subscriptions
func (u *botUseCase) HandleListSubscriptions(ctx context.Context, userID int64) ([]domain.Subscription, error) {
	// TODO: This should be retrieved from subscription service via Kafka
	// For now, return empty list as this is just a skeleton
	u.logger.Info().
		Int64("user_id", userID).
		Msg("Listing subscriptions")

	return []domain.Subscription{}, nil
}

// SendNews sends news to user
func (u *botUseCase) SendNews(ctx context.Context, news *domain.NewsMessage) error {
	u.logger.Info().
		Int64("user_id", news.UserID).
		Str("channel_id", news.ChannelID).
		Msg("Sending news to user")

	var err error
	if len(news.MediaURLs) > 0 {
		err = u.telegramBot.SendMessageWithMedia(ctx, news.UserID, news.Content, news.MediaURLs)
	} else {
		err = u.telegramBot.SendMessage(ctx, news.UserID, news.Content)
	}

	if err != nil {
		u.logger.Error().Err(err).
			Int64("user_id", news.UserID).
			Msg("Failed to send news")
		return domain.ErrMessageDeliveryFailed
	}

	return nil
}
