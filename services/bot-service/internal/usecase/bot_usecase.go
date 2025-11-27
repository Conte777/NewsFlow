package usecase

import (
	"context"

	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
	"github.com/rs/zerolog"
)

type BotUseCase struct {
	kafkaProducer domain.KafkaProducer
	telegramBot   domain.TelegramBot
	logger        zerolog.Logger
}

func NewBotUseCase(kafkaProducer domain.KafkaProducer, telegramBot domain.TelegramBot, logger zerolog.Logger) *BotUseCase {
	return &BotUseCase{
		kafkaProducer: kafkaProducer,
		telegramBot:   telegramBot,
		logger:        logger,
	}
}

// SetTelegramBot позволяет установить TelegramBot после создания use case
func (uc *BotUseCase) SetTelegramBot(telegramBot domain.TelegramBot) {
	uc.telegramBot = telegramBot
}

// Реализуйте остальные методы интерфейса domain.BotUseCase
func (uc *BotUseCase) HandleStart(ctx context.Context, userID int64, chatTitle string) (string, error) {
	// Реализация
	return "Добро пожаловать!", nil
}

func (uc *BotUseCase) HandleHelp(ctx context.Context) (string, error) {
	// Реализация
	return "Справка по командам", nil
}

func (uc *BotUseCase) HandleSubscribe(ctx context.Context, userID int64, channels string) (string, error) {
	// Реализация
	return "Подписка оформлена", nil
}

func (uc *BotUseCase) HandleUnsubscribe(ctx context.Context, userID int64, channels string) (string, error) {
	// Реализация
	return "Отписка выполнена", nil
}

func (uc *BotUseCase) HandleListSubscriptions(ctx context.Context, userID int64) ([]domain.Subscription, error) {
	// Реализация
	return []domain.Subscription{}, nil
}

func (uc *BotUseCase) SendNews(ctx context.Context, news *domain.NewsMessage) error {
	// Реализация отправки новости пользователю
	return nil
}
