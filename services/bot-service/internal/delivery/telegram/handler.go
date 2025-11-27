package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

// TelegramHandler реализует domain.TelegramBot интерфейс
type TelegramHandler struct {
	bot        *tgbot.Bot
	logger     zerolog.Logger
	botUseCase domain.BotUseCase
	running    bool
}

// NewHandler создает новый экземпляр TelegramHandler
func NewHandler(token string, logger zerolog.Logger, botUseCase domain.BotUseCase) (domain.TelegramBot, error) {
	if token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}

	if botUseCase == nil {
		return nil, fmt.Errorf("bot use case is required")
	}

	// Опции для бота
	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(defaultHandler),
	}

	// Создаем бота
	bot, err := tgbot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &TelegramHandler{
		bot:        bot,
		logger:     logger,
		botUseCase: botUseCase,
		running:    false,
	}, nil
}

// Start запускает бота
func (h *TelegramHandler) Start(ctx context.Context) error {
	if h.running {
		return fmt.Errorf("bot is already running")
	}

	// Регистрируем обработчики команд
	if err := h.registerHandlers(); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	h.logger.Info().Msg("Starting Telegram bot...")

	// Запускаем бота
	h.running = true
	h.bot.Start(ctx)
	h.running = false

	h.logger.Info().Msg("Telegram bot stopped")
	return nil
}

// Stop останавливает бота
func (h *TelegramHandler) Stop(ctx context.Context) error {
	if !h.running {
		return fmt.Errorf("bot is not running")
	}

	h.logger.Info().Msg("Stopping Telegram bot...")

	// Создаем контекст с таймаутом для graceful shutdown
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	h.bot.Stop(stopCtx)
	h.running = false

	h.logger.Info().Msg("Telegram bot stopped successfully")
	return nil
}

// registerHandlers регистрирует обработчики команд
func (h *TelegramHandler) registerHandlers() error {
	// Регистрируем все команды согласно требованиям
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, h.handleStart)
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/help", tgbot.MatchTypeExact, h.handleHelp)
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/subscribe", tgbot.MatchTypePrefix, h.handleSubscribe)
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/unsubscribe", tgbot.MatchTypePrefix, h.handleUnsubscribe)
	h.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/list", tgbot.MatchTypeExact, h.handleList)

	h.logger.Info().Msg("All command handlers registered successfully")
	return nil
}

// handleStart обрабатывает команду /start
func (h *TelegramHandler) handleStart(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	// Логируем команду
	h.logCommand(userID, "/start", "processing")

	// Вызываем use case
	result, err := h.botUseCase.HandleStart(ctx, int64(userID), chatID)
	if err != nil {
		h.logError(userID, "/start", err)
		h.sendMessage(ctx, chatID, "❌ Произошла ошибка при обработке команды /start")
		return
	}

	h.sendMessage(ctx, chatID, result)
	h.logCommand(userID, "/start", "success")
}

// handleHelp обрабатывает команду /help
func (h *TelegramHandler) handleHelp(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(userID, "/help", "processing")

	result, err := h.botUseCase.HandleHelp(ctx, int64(userID))
	if err != nil {
		h.logError(userID, "/help", err)
		h.sendMessage(ctx, chatID, "❌ Произошла ошибка при обработке команды /help")
		return
	}

	h.sendMessage(ctx, chatID, result)
	h.logCommand(userID, "/help", "success")
}

// handleSubscribe обрабатывает команду /subscribe
func (h *TelegramHandler) handleSubscribe(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	h.logCommand(userID, "/subscribe", "processing")

	// Парсим аргументы команды
	channels, err := h.parseChannels(text, "/subscribe")
	if err != nil {
		h.logError(userID, "/subscribe", err)
		h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка парсинга: %s", err.Error()))
		return
	}

	if len(channels) == 0 {
		h.sendMessage(ctx, chatID, "❌ Укажите каналы для подписки. Пример: /subscribe @channel1 @channel2")
		return
	}

	result, err := h.botUseCase.HandleSubscribe(ctx, int64(userID), channels)
	if err != nil {
		h.logError(userID, "/subscribe", err)
		h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка подписки: %s", err.Error()))
		return
	}

	h.sendMessage(ctx, chatID, result)
	h.logCommand(userID, "/subscribe", fmt.Sprintf("subscribed to %v", channels))
}

// handleUnsubscribe обрабатывает команду /unsubscribe
func (h *TelegramHandler) handleUnsubscribe(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	h.logCommand(userID, "/unsubscribe", "processing")

	// Парсим аргументы команды
	channels, err := h.parseChannels(text, "/unsubscribe")
	if err != nil {
		h.logError(userID, "/unsubscribe", err)
		h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка парсинга: %s", err.Error()))
		return
	}

	if len(channels) == 0 {
		h.sendMessage(ctx, chatID, "❌ Укажите каналы для отписки. Пример: /unsubscribe @channel1 @channel2")
		return
	}

	result, err := h.botUseCase.HandleUnsubscribe(ctx, int64(userID), channels)
	if err != nil {
		h.logError(userID, "/unsubscribe", err)
		h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка отписки: %s", err.Error()))
		return
	}

	h.sendMessage(ctx, chatID, result)
	h.logCommand(userID, "/unsubscribe", fmt.Sprintf("unsubscribed from %v", channels))
}

// handleList обрабатывает команду /list
func (h *TelegramHandler) handleList(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(userID, "/list", "processing")

	result, err := h.botUseCase.HandleListSubscriptions(ctx, int64(userID))
	if err != nil {
		h.logError(userID, "/list", err)
		h.sendMessage(ctx, chatID, "❌ Произошла ошибка при получении списка подписок")
		return
	}

	h.sendMessage(ctx, chatID, result)
	h.logCommand(userID, "/list", "success")
}

// parseChannels парсит и валидирует каналы из аргументов команды
func (h *TelegramHandler) parseChannels(text, command string) ([]string, error) {
	// Убираем команду из текста
	args := strings.TrimSpace(strings.TrimPrefix(text, command))
	if args == "" {
		return nil, nil
	}

	// Разделяем аргументы по пробелам
	rawChannels := strings.Fields(args)
	validChannels := make([]string, 0, len(rawChannels))

	for _, channel := range rawChannels {
		// Валидируем формат канала (должен начинаться с @)
		if !strings.HasPrefix(channel, "@") {
			return nil, fmt.Errorf("неверный формат канала '%s'. Канал должен начинаться с @", channel)
		}

		// Проверяем длину канала (без @)
		channelName := strings.TrimPrefix(channel, "@")
		if len(channelName) == 0 {
			return nil, fmt.Errorf("неверный формат канала '%s'. Укажите название канала после @", channel)
		}

		// Проверяем валидность символов в названии канала
		if !isValidChannelName(channelName) {
			return nil, fmt.Errorf("неверные символы в названии канала '%s'. Допустимы только буквы, цифры и подчеркивания", channel)
		}

		validChannels = append(validChannels, channel)
	}

	return validChannels, nil
}

// isValidChannelName проверяет валидность названия канала
func isValidChannelName(name string) bool {
	for _, char := range name {
		if !isValidChannelChar(char) {
			return false
		}
	}
	return true
}

// isValidChannelChar проверяет валидность символа в названии канала
func isValidChannelChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

// sendMessage отправляет сообщение в чат
func (h *TelegramHandler) sendMessage(ctx context.Context, chatID int64, text string) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := h.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})

	if err != nil {
		h.logger.Error().
			Int64("chat_id", chatID).
			Err(err).
			Msg("Failed to send Telegram message")
	}
}

// logCommand логирует успешные команды
func (h *TelegramHandler) logCommand(userID int, command, result string) {
	h.logger.Info().
		Int("user_id", userID).
		Str("command", command).
		Str("result", result).
		Msg("Telegram command processed")
}

// logError логирует ошибки команд
func (h *TelegramHandler) logError(userID int, command string, err error) {
	h.logger.Error().
		Int("user_id", userID).
		Str("command", command).
		Err(err).
		Msg("Telegram command failed")
}

// defaultHandler обрабатывает сообщения без команд
func defaultHandler(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	// Отвечаем на сообщения без команд
	_, err := bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "🤖 Используйте команды для взаимодействия с ботом. Напишите /help для списка доступных команд.",
	})

	if err != nil {
		// Логируем ошибку, но не прерываем выполнение
	}
}
