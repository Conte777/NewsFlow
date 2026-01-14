package telegram

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
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
	httpClient *http.Client
}

// Константы для Telegram API
const (
	MaxMessageLength    = 4096
	MessageSplitTimeout = 2 * time.Second
	RequestTimeout      = 30 * time.Second
	MaxMediaGroupSize   = 10
	MaxRetries          = 3
	RetryDelay          = 2 * time.Second
	MaxFileSize         = 50 * 1024 * 1024 // 50MB - лимит Telegram для файлов
	MaxPhotoSize        = 10 * 1024 * 1024 // 10MB - лимит для фото
	MaxVideoSize        = 50 * 1024 * 1024 // 50MB - лимит для видео
)

// MediaType представляет тип медиа файла
type MediaType string

const (
	MediaTypePhoto       MediaType = "photo"
	MediaTypeVideo       MediaType = "video"
	MediaTypeDocument    MediaType = "document"
	MediaTypeUnsupported MediaType = "unsupported"
)

// MediaInfo содержит информацию о медиа файле
type MediaInfo struct {
	URL      string
	Type     MediaType
	MimeType string
	FileName string
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
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
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
				parts = append(parts, splitLines...)
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

// SendMessageWithMedia отправляет сообщение с медиа файлами пользователю
func (h *TelegramHandler) SendMessageWithMedia(ctx context.Context, userID int64, text string, mediaURLs []string) error {
	if len(mediaURLs) == 0 {
		return h.SendMessage(ctx, userID, text)
	}

	h.logger.Info().
		Int64("user_id", userID).
		Int("media_count", len(mediaURLs)).
		Msg("Sending message with media")

	// Валидируем и классифицируем медиа файлы
	mediaInfos, err := h.validateAndClassifyMedia(mediaURLs)
	if err != nil {
		return fmt.Errorf("media validation failed: %w", err)
	}

	// Отправляем в зависимости от количества медиа
	if len(mediaInfos) == 1 {
		return h.sendSingleMedia(ctx, userID, text, mediaInfos[0])
	} else if len(mediaInfos) <= MaxMediaGroupSize {
		return h.sendMediaGroup(ctx, userID, text, mediaInfos)
	} else {
		return h.sendMultipleMediaGroups(ctx, userID, text, mediaInfos)
	}
}

// validateAndClassifyMedia валидирует URL и определяет тип медиа
func (h *TelegramHandler) validateAndClassifyMedia(mediaURLs []string) ([]MediaInfo, error) {
	var mediaInfos []MediaInfo

	for _, mediaURL := range mediaURLs {
		// Валидируем URL
		if err := h.validateMediaURL(mediaURL); err != nil {
			return nil, err
		}

		// Определяем тип медиа
		mediaInfo, err := h.classifyMedia(mediaURL)
		if err != nil {
			return nil, err
		}

		// Проверяем размер файла
		if err := h.checkFileSize(mediaInfo); err != nil {
			return nil, err
		}

		mediaInfos = append(mediaInfos, mediaInfo)
	}

	return mediaInfos, nil
}

// validateMediaURL валидирует URL медиа файла
func (h *TelegramHandler) validateMediaURL(mediaURL string) error {
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return fmt.Errorf("invalid URL format '%s': %w", mediaURL, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme '%s' for '%s'", parsedURL.Scheme, mediaURL)
	}

	// Проверяем что URL доступен (HEAD запрос)
	req, err := http.NewRequest("HEAD", mediaURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request for '%s': %w", mediaURL, err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to access media URL '%s': %w", mediaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("media URL '%s' returned status %d", mediaURL, resp.StatusCode)
	}

	return nil
}

// classifyMedia определяет тип медиа файла по URL
func (h *TelegramHandler) classifyMedia(mediaURL string) (MediaInfo, error) {
	parsedURL, _ := url.Parse(mediaURL)
	fileName := path.Base(parsedURL.Path)
	ext := strings.ToLower(path.Ext(fileName))

	// Определяем MIME тип по расширению
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Если не определили по расширению, пытаемся получить из URL
		req, err := http.NewRequest("HEAD", mediaURL, nil)
		if err == nil {
			resp, err := h.httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				contentType := resp.Header.Get("Content-Type")
				if contentType != "" {
					mimeType = contentType
				}
			}
		}
	}

	// Классифицируем по MIME типу или расширению
	mediaType := h.determineMediaType(mimeType, ext)

	return MediaInfo{
		URL:      mediaURL,
		Type:     mediaType,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}

// determineMediaType определяет тип медиа по MIME типу и расширению
func (h *TelegramHandler) determineMediaType(mimeType, ext string) MediaType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return MediaTypePhoto
	case strings.HasPrefix(mimeType, "video/"):
		return MediaTypeVideo
	case strings.HasPrefix(mimeType, "application/") || strings.HasPrefix(mimeType, "text/"):
		return MediaTypeDocument
	}

	// Если MIME тип не определился, пробуем по расширению
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return MediaTypePhoto
	case ".mp4", ".avi", ".mov", ".mkv", ".webm":
		return MediaTypeVideo
	case ".pdf", ".doc", ".docx", ".txt", ".zip", ".rar":
		return MediaTypeDocument
	default:
		return MediaTypeUnsupported
	}
}

// checkFileSize проверяет размер файла в соответствии с лимитами Telegram
func (h *TelegramHandler) checkFileSize(mediaInfo MediaInfo) error {
	// Получаем размер файла через HEAD запрос
	req, err := http.NewRequest("HEAD", mediaInfo.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get file size for '%s': %w", mediaInfo.URL, err)
	}
	defer resp.Body.Close()

	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		h.logger.Warn().
			Str("url", mediaInfo.URL).
			Msg("Could not determine file size, proceeding anyway")
		return nil
	}

	var fileSize int64
	fmt.Sscanf(contentLength, "%d", &fileSize)

	// Проверяем лимиты в зависимости от типа медиа
	switch mediaInfo.Type {
	case MediaTypePhoto:
		if fileSize > MaxPhotoSize {
			return fmt.Errorf("photo size %d bytes exceeds limit %d bytes", fileSize, MaxPhotoSize)
		}
	case MediaTypeVideo:
		if fileSize > MaxVideoSize {
			return fmt.Errorf("video size %d bytes exceeds limit %d bytes", fileSize, MaxVideoSize)
		}
	case MediaTypeDocument:
		if fileSize > MaxFileSize {
			return fmt.Errorf("document size %d bytes exceeds limit %d bytes", fileSize, MaxFileSize)
		}
	}

	return nil
}

// sendSingleMedia отправляет одно медиа с текстом
func (h *TelegramHandler) sendSingleMedia(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
	h.logger.Debug().
		Int64("user_id", userID).
		Str("media_type", string(mediaInfo.Type)).
		Str("url", mediaInfo.URL).
		Msg("Sending single media")

	var err error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		switch mediaInfo.Type {
		case MediaTypePhoto:
			err = h.sendPhoto(ctx, userID, text, mediaInfo)
		case MediaTypeVideo:
			err = h.sendVideo(ctx, userID, text, mediaInfo)
		case MediaTypeDocument:
			err = h.sendDocument(ctx, userID, text, mediaInfo)
		default:
			return fmt.Errorf("unsupported media type: %s", mediaInfo.Type)
		}

		if err == nil {
			break
		}

		h.logger.Warn().
			Int64("user_id", userID).
			Int("attempt", attempt).
			Err(err).
			Msg("Failed to send media, retrying")

		if attempt < MaxRetries {
			time.Sleep(RetryDelay * time.Duration(attempt))
		}
	}

	if err != nil {
		h.logMediaSend(userID, 1, false, err)
		return fmt.Errorf("failed to send media after %d attempts: %w", MaxRetries, err)
	}

	h.logMediaSend(userID, 1, true, nil)
	return nil
}

// sendPhoto отправляет фото
func (h *TelegramHandler) sendPhoto(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.SendPhoto(msgCtx, &tgbot.SendPhotoParams{
		ChatID:    userID,
		Photo:     &models.InputFileString{Data: mediaInfo.URL},
		Caption:   text,
		ParseMode: models.ParseModeHTML,
	})

	return err
}

// sendVideo отправляет видео
func (h *TelegramHandler) sendVideo(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.SendVideo(msgCtx, &tgbot.SendVideoParams{
		ChatID:    userID,
		Video:     &models.InputFileString{Data: mediaInfo.URL},
		Caption:   text,
		ParseMode: models.ParseModeHTML,
	})

	return err
}

// sendDocument отправляет документ
func (h *TelegramHandler) sendDocument(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.SendDocument(msgCtx, &tgbot.SendDocumentParams{
		ChatID:    userID,
		Document:  &models.InputFileString{Data: mediaInfo.URL},
		Caption:   text,
		ParseMode: models.ParseModeHTML,
	})

	return err
}

// sendMediaGroup отправляет группу медиа (2-10 файлов)
func (h *TelegramHandler) sendMediaGroup(ctx context.Context, userID int64, text string, mediaInfos []MediaInfo) error {
	h.logger.Debug().
		Int64("user_id", userID).
		Int("media_count", len(mediaInfos)).
		Msg("Sending media group")

	// Для версии библиотеки, где InputMedia не поддерживается,
	// отправляем медиа по отдельности с задержкой
	if len(mediaInfos) == 1 {
		return h.sendSingleMedia(ctx, userID, text, mediaInfos[0])
	}

	// Отправляем первое медиа с текстом
	if err := h.sendSingleMedia(ctx, userID, text, mediaInfos[0]); err != nil {
		return fmt.Errorf("failed to send first media: %w", err)
	}

	// Отправляем остальные медиа без текста
	for i := 1; i < len(mediaInfos); i++ {
		if err := h.sendSingleMedia(ctx, userID, "", mediaInfos[i]); err != nil {
			h.logger.Error().
				Int64("user_id", userID).
				Int("media_index", i).
				Err(err).
				Msg("Failed to send media in group")
			// Продолжаем отправлять остальные медиа, даже если одно не удалось
		}

		// Задержка между отправками
		if i < len(mediaInfos)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	h.logMediaSend(userID, len(mediaInfos), true, nil)
	return nil
}

// sendMultipleMediaGroups отправляет несколько групп медиа (более 10 файлов)
func (h *TelegramHandler) sendMultipleMediaGroups(ctx context.Context, userID int64, text string, mediaInfos []MediaInfo) error {
	h.logger.Info().
		Int64("user_id", userID).
		Int("total_media", len(mediaInfos)).
		Msg("Sending multiple media groups")

	// Разбиваем на группы по MaxMediaGroupSize
	var groups [][]MediaInfo
	for i := 0; i < len(mediaInfos); i += MaxMediaGroupSize {
		end := i + MaxMediaGroupSize
		if end > len(mediaInfos) {
			end = len(mediaInfos)
		}
		groups = append(groups, mediaInfos[i:end])
	}

	totalGroups := len(groups)
	successCount := 0

	// Отправляем первую группу с текстом
	if err := h.sendMediaGroup(ctx, userID, text, groups[0]); err != nil {
		h.logger.Error().
			Int64("user_id", userID).
			Int("group", 1).
			Err(err).
			Msg("Failed to send first media group")
	} else {
		successCount++
	}

	// Отправляем остальные группы без текста (или с индикатором прогресса)
	for i := 1; i < totalGroups; i++ {
		groupText := ""
		if totalGroups > 1 {
			groupText = fmt.Sprintf("<i>(Медиа %d/%d)</i>", i+1, totalGroups)
		}

		if err := h.sendMediaGroup(ctx, userID, groupText, groups[i]); err != nil {
			h.logger.Error().
				Int64("user_id", userID).
				Int("group", i+1).
				Err(err).
				Msg("Failed to send media group")
		} else {
			successCount++
		}

		// Задержка между группами
		if i < totalGroups-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	h.logger.Info().
		Int64("user_id", userID).
		Int("success_groups", successCount).
		Int("total_groups", totalGroups).
		Msg("Finished sending multiple media groups")

	if successCount == 0 {
		return fmt.Errorf("failed to send all media groups")
	}

	if successCount < totalGroups {
		return fmt.Errorf("sent only %d out of %d media groups", successCount, totalGroups)
	}

	return nil
}

// logMediaSend логирует результат отправки медиа
func (h *TelegramHandler) logMediaSend(userID int64, mediaCount int, success bool, err error) {
	logEvent := h.logger.Info()
	if !success {
		logEvent = h.logger.Error()
	}

	logEvent.
		Int64("user_id", userID).
		Int("media_count", mediaCount).
		Bool("success", success)

	if err != nil {
		logEvent.Err(err)
	}

	logEvent.Msg("Media send attempt completed")
}

// handleMediaSendError обрабатывает ошибки отправки медиа
func (h *TelegramHandler) handleMediaSendError(userID int64, mediaCount int, err error) error {
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "wrong file identifier") || strings.Contains(errorMsg, "failed to get HTTP URL content"):
		h.logger.Warn().
			Int64("user_id", userID).
			Int("media_count", mediaCount).
			Msg("Invalid media URL or file not accessible")
		return fmt.Errorf("invalid media URL or file not accessible")

	case strings.Contains(errorMsg, "file is too big"):
		h.logger.Warn().
			Int64("user_id", userID).
			Int("media_count", mediaCount).
			Msg("File size exceeds Telegram limits")
		return fmt.Errorf("file size exceeds Telegram limits")

	case strings.Contains(errorMsg, "wrong type of the web page content"):
		h.logger.Warn().
			Int64("user_id", userID).
			Int("media_count", mediaCount).
			Msg("Unsupported media type")
		return fmt.Errorf("unsupported media type")

	case strings.Contains(errorMsg, "Too Many Requests"):
		h.logger.Warn().
			Int64("user_id", userID).
			Int("media_count", mediaCount).
			Msg("Rate limit exceeded for media sending")
		return fmt.Errorf("rate limit exceeded, please try again later")

	case strings.Contains(errorMsg, "network error"), strings.Contains(errorMsg, "timeout"):
		h.logger.Warn().
			Int64("user_id", userID).
			Int("media_count", mediaCount).
			Msg("Network error while sending media")
		return fmt.Errorf("network error, please try again")

	default:
		h.logger.Error().
			Int64("user_id", userID).
			Int("media_count", mediaCount).
			Err(err).
			Msg("Unknown error while sending media")
		return fmt.Errorf("failed to send media: %w", err)
	}
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
