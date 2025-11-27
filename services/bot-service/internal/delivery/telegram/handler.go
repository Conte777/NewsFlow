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
}

// Константы для Telegram API
const (
	MaxMessageLength    = 4096
	MessageSplitTimeout = 2 * time.Second
	RequestTimeout      = 10 * time.Second
)

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
	}, nil
}

// Start запускает бота
func (h *TelegramHandler) Start(ctx context.Context) error {
	// Регистрируем обработчики команд
	if err := h.registerHandlers(); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	h.logger.Info().Msg("Starting Telegram bot...")

	// Запускаем бота (блокирующий вызов)
	h.bot.Start(ctx)

	h.logger.Info().Msg("Telegram bot stopped")
	return nil
}

// Stop останавливает бота
func (h *TelegramHandler) Stop() error {
	h.logger.Info().Msg("Stopping Telegram bot...")
	// В этой версии библиотеки бот останавливается через контекст
	// в методе Start, поэтому здесь просто логируем
	return nil
}

// SendMessage отправляет текстовое сообщение пользователю с поддержкой HTML форматирования
func (h *TelegramHandler) SendMessage(ctx context.Context, userID int64, text string) error {
	if text == "" {
		h.logger.Warn().
			Int64("user_id", userID).
			Msg("Attempt to send empty message")
		return fmt.Errorf("message text cannot be empty")
	}

	h.logger.Debug().
		Int64("user_id", userID).
		Int("text_length", len(text)).
		Msg("Sending message to user")

	// Если сообщение слишком длинное, разбиваем на части
	if len(text) > MaxMessageLength {
		return h.sendSplitMessage(ctx, userID, text)
	}

	return h.sendSingleMessage(ctx, userID, text)
}

// sendSingleMessage отправляет одно сообщение
func (h *TelegramHandler) sendSingleMessage(ctx context.Context, userID int64, text string) error {
	// Создаем контекст с таймаутом
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.SendMessage(msgCtx, &tgbot.SendMessageParams{
		ChatID:    userID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})

	if err != nil {
		// Обрабатываем различные типы ошибок
		handledErr := h.handleSendMessageError(userID, err)
		h.logMessageSend(userID, len(text), false, handledErr)
		return handledErr
	}

	h.logMessageSend(userID, len(text), true, nil)
	return nil
}

// sendSplitMessage разбивает длинное сообщение на части и отправляет их
func (h *TelegramHandler) sendSplitMessage(ctx context.Context, userID int64, text string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Int("total_length", len(text)).
		Msg("Splitting long message into parts")

	parts := h.splitMessage(text)
	totalParts := len(parts)
	successCount := 0

	for i, part := range parts {
		partNumber := i + 1

		// Добавляем индикатор прогресса для частей
		if totalParts > 1 {
			part = fmt.Sprintf("<i>(Часть %d/%d)</i>\n\n%s", partNumber, totalParts, part)
		}

		err := h.sendSingleMessage(ctx, userID, part)
		if err != nil {
			h.logger.Error().
				Int64("user_id", userID).
				Int("part", partNumber).
				Int("total_parts", totalParts).
				Err(err).
				Msg("Failed to send message part")

			// Продолжаем отправлять остальные части, даже если одна не удалась
			continue
		}

		successCount++

		// Добавляем небольшую задержку между отправками, чтобы не превысить лимиты Telegram
		if partNumber < totalParts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(MessageSplitTimeout):
				// Продолжаем отправку
			}
		}
	}

	h.logger.Info().
		Int64("user_id", userID).
		Int("success_parts", successCount).
		Int("total_parts", totalParts).
		Msg("Finished sending split message")

	if successCount == 0 {
		return fmt.Errorf("failed to send all message parts")
	}

	if successCount < totalParts {
		return fmt.Errorf("sent only %d out of %d message parts", successCount, totalParts)
	}

	return nil
}

// splitMessage разбивает текст на части, не превышающие максимальную длину
func (h *TelegramHandler) splitMessage(text string) []string {
	if len(text) <= MaxMessageLength {
		return []string{text}
	}

	var parts []string
	lines := strings.Split(text, "\n")
	currentPart := strings.Builder{}
	currentLength := 0

	for _, line := range lines {
		lineLength := len(line) + 1 // +1 для символа новой строки

		// Если добавление этой строки превысит лимит, начинаем новую часть
		if currentLength+lineLength > MaxMessageLength {
			if currentPart.Len() > 0 {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
				currentLength = 0
			}

			// Если одна строка сама по себе слишком длинная, разбиваем её
			if lineLength > MaxMessageLength {
				splitLines := h.splitLongLine(line)
				for _, splitLine := range splitLines {
					parts = append(parts, splitLine)
				}
				continue
			}
		}

		// Добавляем строку к текущей части
		if currentPart.Len() > 0 {
			currentPart.WriteString("\n")
			currentLength++
		}
		currentPart.WriteString(line)
		currentLength += len(line)
	}

	// Добавляем последнюю часть, если она не пустая
	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	return parts
}

// splitLongLine разбивает очень длинную строку на части
func (h *TelegramHandler) splitLongLine(line string) []string {
	if len(line) <= MaxMessageLength {
		return []string{line}
	}

	var parts []string
	start := 0

	for start < len(line) {
		end := start + MaxMessageLength
		if end > len(line) {
			end = len(line)
		}

		// Пытаемся разбить по границе слова
		if end < len(line) {
			// Ищем последний пробел в пределах части
			lastSpace := strings.LastIndex(line[start:end], " ")
			if lastSpace > 0 {
				end = start + lastSpace
			}
		}

		parts = append(parts, line[start:end])
		start = end

		// Пропускаем пробелы в начале следующей части
		for start < len(line) && line[start] == ' ' {
			start++
		}
	}

	return parts
}

// handleSendMessageError обрабатывает ошибки отправки сообщений
func (h *TelegramHandler) handleSendMessageError(userID int64, err error) error {
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "Forbidden"):
		h.logger.Warn().
			Int64("user_id", userID).
			Msg("User blocked the bot or chat not found")
		return fmt.Errorf("user blocked the bot or chat not found")

	case strings.Contains(errorMsg, "Bad Request: chat not found"):
		h.logger.Warn().
			Int64("user_id", userID).
			Msg("Chat not found")
		return fmt.Errorf("chat not found")

	case strings.Contains(errorMsg, "Too Many Requests"):
		h.logger.Warn().
			Int64("user_id", userID).
			Msg("Rate limit exceeded")
		return fmt.Errorf("rate limit exceeded, please try again later")

	case strings.Contains(errorMsg, "network error"), strings.Contains(errorMsg, "timeout"):
		h.logger.Warn().
			Int64("user_id", userID).
			Msg("Network error while sending message")
		return fmt.Errorf("network error, please try again")

	default:
		h.logger.Error().
			Int64("user_id", userID).
			Err(err).
			Msg("Unknown error while sending message")
		return fmt.Errorf("failed to send message: %w", err)
	}
}

// logMessageSend логирует результат отправки сообщения
func (h *TelegramHandler) logMessageSend(userID int64, length int, success bool, err error) {
	logEvent := h.logger.Info()
	if !success {
		logEvent = h.logger.Error()
	}

	logEvent.
		Int64("user_id", userID).
		Int("message_length", length).
		Bool("success", success)

	if err != nil {
		logEvent.Err(err)
	}

	logEvent.Msg("Message send attempt completed")
}

// SendMessageWithMedia отправляет сообщение с медиа пользователю
func (h *TelegramHandler) SendMessageWithMedia(ctx context.Context, userID int64, text string, mediaURLs []string) error {
	h.logger.Info().
		Int64("user_id", userID).
		Int("media_count", len(mediaURLs)).
		Msg("Sending message with media")

	// Если есть медиа, добавляем информацию о них в текст
	if len(mediaURLs) > 0 {
		mediaInfo := fmt.Sprintf("\n\n<code>📎 Прикреплено медиа файлов: %d</code>", len(mediaURLs))

		// Показываем первые несколько URL
		maxUrlsToShow := 3
		for i, url := range mediaURLs {
			if i >= maxUrlsToShow {
				mediaInfo += fmt.Sprintf("\n<code>... и ещё %d</code>", len(mediaURLs)-maxUrlsToShow)
				break
			}
			// Обрезаем длинные URL для лучшего отображения
			if len(url) > 50 {
				url = url[:47] + "..."
			}
			mediaInfo += fmt.Sprintf("\n<code>• %s</code>", url)
		}

		text += mediaInfo
	}

	return h.SendMessage(ctx, userID, text)
}

// ===== ОСТАЛЬНЫЕ МЕТОДЫ (без изменений) =====

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
	h.logCommand(int64(userID), "/start", "processing")

	// Вызываем use case
	result, err := h.botUseCase.HandleStart(ctx, int64(userID), update.Message.Chat.Title)
	if err != nil {
		h.logError(int64(userID), "/start", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при обработке команды /start")
		return
	}

	h.sendResponse(ctx, chatID, result)
	h.logCommand(int64(userID), "/start", "success")
}

// handleHelp обрабатывает команду /help
func (h *TelegramHandler) handleHelp(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(int64(userID), "/help", "processing")

	result, err := h.botUseCase.HandleHelp(ctx)
	if err != nil {
		h.logError(int64(userID), "/help", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при обработке команды /help")
		return
	}

	h.sendResponse(ctx, chatID, result)
	h.logCommand(int64(userID), "/help", "success")
}

// handleSubscribe обрабатывает команду /subscribe
func (h *TelegramHandler) handleSubscribe(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	h.logCommand(int64(userID), "/subscribe", "processing")

	// Парсим аргументы команды
	channels, err := h.parseChannels(text, "/subscribe")
	if err != nil {
		h.logError(int64(userID), "/subscribe", err)
		h.sendResponse(ctx, chatID, fmt.Sprintf("❌ Ошибка парсинга: %s", err.Error()))
		return
	}

	if len(channels) == 0 {
		h.sendResponse(ctx, chatID, "❌ Укажите каналы для подписки. Пример: /subscribe @channel1 @channel2")
		return
	}

	// Преобразуем []string в string (как ожидает use case)
	channelsStr := strings.Join(channels, " ")

	result, err := h.botUseCase.HandleSubscribe(ctx, int64(userID), channelsStr)
	if err != nil {
		h.logError(int64(userID), "/subscribe", err)
		h.sendResponse(ctx, chatID, fmt.Sprintf("❌ Ошибка подписки: %s", err.Error()))
		return
	}

	h.sendResponse(ctx, chatID, result)
	h.logCommand(int64(userID), "/subscribe", fmt.Sprintf("subscribed to %v", channels))
}

// handleUnsubscribe обрабатывает команду /unsubscribe
func (h *TelegramHandler) handleUnsubscribe(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	h.logCommand(int64(userID), "/unsubscribe", "processing")

	// Парсим аргументы команды
	channels, err := h.parseChannels(text, "/unsubscribe")
	if err != nil {
		h.logError(int64(userID), "/unsubscribe", err)
		h.sendResponse(ctx, chatID, fmt.Sprintf("❌ Ошибка парсинга: %s", err.Error()))
		return
	}

	if len(channels) == 0 {
		h.sendResponse(ctx, chatID, "❌ Укажите каналы для отписки. Пример: /unsubscribe @channel1 @channel2")
		return
	}

	// Преобразуем []string в string (как ожидает use case)
	channelsStr := strings.Join(channels, " ")

	result, err := h.botUseCase.HandleUnsubscribe(ctx, int64(userID), channelsStr)
	if err != nil {
		h.logError(int64(userID), "/unsubscribe", err)
		h.sendResponse(ctx, chatID, fmt.Sprintf("❌ Ошибка отписки: %s", err.Error()))
		return
	}

	h.sendResponse(ctx, chatID, result)
	h.logCommand(int64(userID), "/unsubscribe", fmt.Sprintf("unsubscribed from %v", channels))
}

// handleList обрабатывает команду /list
func (h *TelegramHandler) handleList(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(int64(userID), "/list", "processing")

	// Получаем результат как строку
	subscriptions, err := h.botUseCase.HandleListSubscriptions(ctx, int64(userID))
	if err != nil {
		h.logError(int64(userID), "/list", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при получении списка подписок")
		return
	}

	// Форматируем результат как строку
	result := h.formatSubscriptions(subscriptions)
	h.sendResponse(ctx, chatID, result)
	h.logCommand(int64(userID), "/list", "success")
}

// formatSubscriptions форматирует подписки в строку
func (h *TelegramHandler) formatSubscriptions(subscriptions []domain.Subscription) string {
	if len(subscriptions) == 0 {
		return "📋 У вас пока нет подписок"
	}

	var result strings.Builder
	result.WriteString("📋 <b>Ваши подписки:</b>\n")

	for _, sub := range subscriptions {
		result.WriteString(fmt.Sprintf("• <code>%s</code>\n", sub.ChannelName))
	}

	return result.String()
}

// sendResponse отправляет ответное сообщение в чат (внутренний метод)
func (h *TelegramHandler) sendResponse(ctx context.Context, chatID int64, text string) {
	// Используем общий метод SendMessage для отправки ответов
	if err := h.SendMessage(ctx, chatID, text); err != nil {
		h.logger.Error().
			Int64("chat_id", chatID).
			Err(err).
			Msg("Failed to send Telegram response")
	}
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

// logCommand логирует успешные команды
func (h *TelegramHandler) logCommand(userID int64, command, result string) {
	h.logger.Info().
		Int64("user_id", userID).
		Str("command", command).
		Str("result", result).
		Msg("Telegram command processed")
}

// logError логирует ошибки команд
func (h *TelegramHandler) logError(userID int64, command string, err error) {
	h.logger.Error().
		Int64("user_id", userID).
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
