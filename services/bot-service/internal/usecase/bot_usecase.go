package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
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
	// Validate channel ID
	if channelID == "" {
		return "", domain.ErrInvalidChannelID
	}

	// Validate channel format (should start with @)
	if !strings.HasPrefix(channelID, "@") {
		return "", fmt.Errorf("channel ID must start with @: %w", domain.ErrInvalidChannelID)
	}

	// Create subscription
	subscription := &domain.Subscription{
		UserID:      userID,
		ChannelID:   channelID,
		ChannelName: u.extractChannelName(channelID),
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
	// Validate channel ID
	if channelID == "" {
		return "", domain.ErrInvalidChannelID
	}

	// Validate channel format
	if !strings.HasPrefix(channelID, "@") {
		return "", fmt.Errorf("channel ID must start with @: %w", domain.ErrInvalidChannelID)
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

	// Temporary implementation - always return empty list
	// In the future, this will make a request to subscription service
	return []domain.Subscription{}, nil
}

// SendNews sends news to user
func (u *botUseCase) SendNews(ctx context.Context, news *domain.NewsMessage) error { // ← ИСПРАВЛЕНО: добавил *
	u.logger.Info().
		Int64("user_id", news.UserID).
		Str("channel_id", news.ChannelID).
		Str("news_id", news.ID).
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
			Str("news_id", news.ID).
			Msg("Failed to send news")
		return fmt.Errorf("failed to send news: %w", err)
	}

	u.logger.Info().
		Int64("user_id", news.UserID).
		Str("news_id", news.ID).
		Msg("News sent successfully")

	return nil
}

// extractChannelName extracts channel name from channel ID
// For example: "@news_channel" -> "News Channel"
func (u *botUseCase) extractChannelName(channelID string) string {
	if len(channelID) <= 1 {
		return channelID
	}

	// Remove @ and capitalize words
	name := strings.TrimPrefix(channelID, "@")
	name = strings.ReplaceAll(name, "_", " ")

	// Simple capitalization (for Go 1.21+)
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
	}

	return name
}

// ValidateChannel validates channel ID format
func (u *botUseCase) ValidateChannel(channelID string) error {
	if channelID == "" {
		return domain.ErrInvalidChannelID
	}

	if !strings.HasPrefix(channelID, "@") {
		return fmt.Errorf("channel ID must start with @: %w", domain.ErrInvalidChannelID)
	}

	if len(channelID) < 2 {
		return fmt.Errorf("channel ID too short: %w", domain.ErrInvalidChannelID)
	}

	return nil
}
