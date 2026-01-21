// Package telegram contains Telegram delivery handlers
package telegram

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/dto"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/usecase/buissines"
)

// Constants for Telegram API
const (
	MaxMessageLength    = 4096
	MessageSplitTimeout = 2 * time.Second
	RequestTimeout      = 30 * time.Second
	DownloadTimeout     = 120 * time.Second // Timeout for downloading files from S3
	MaxMediaGroupSize   = 10
	MaxRetries          = 3
	RetryDelay          = 2 * time.Second
	MaxFileSize         = 50 * 1024 * 1024 // 50MB
	MaxPhotoSize        = 10 * 1024 * 1024 // 10MB
	MaxVideoSize        = 50 * 1024 * 1024 // 50MB
)

// MediaType represents media file type
type MediaType string

const (
	MediaTypePhoto       MediaType = "photo"
	MediaTypeVideo       MediaType = "video"
	MediaTypeDocument    MediaType = "document"
	MediaTypeUnsupported MediaType = "unsupported"
)

// MediaInfo contains media file information
type MediaInfo struct {
	URL      string
	Type     MediaType
	MimeType string
	FileName string
}

// Handlers contains Telegram command handlers
// Implements deps.TelegramSender interface
type Handlers struct {
	uc         *buissines.UseCase
	bot        *tgbot.Bot
	logger     zerolog.Logger
	httpClient *http.Client
}

// NewHandlers creates new Telegram handlers
func NewHandlers(uc *buissines.UseCase, bot *tgbot.Bot, logger zerolog.Logger) *Handlers {
	return &Handlers{
		uc:     uc,
		bot:    bot,
		logger: logger,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

// SendMessage implements deps.TelegramSender interface
func (h *Handlers) SendMessage(ctx context.Context, userID int64, text string) error {
	if text == "" {
		h.logger.Warn().Int64("user_id", userID).Msg("Attempt to send empty message")
		return fmt.Errorf("message text cannot be empty")
	}

	h.logger.Debug().Int64("user_id", userID).Int("text_length", len(text)).Msg("Sending message to user")

	if len(text) > MaxMessageLength {
		return h.sendSplitMessage(ctx, userID, text)
	}

	return h.sendSingleMessage(ctx, userID, text)
}

// SendMessageWithMedia implements deps.TelegramSender interface
func (h *Handlers) SendMessageWithMedia(ctx context.Context, userID int64, text string, mediaURLs []string) error {
	if len(mediaURLs) == 0 {
		return h.SendMessage(ctx, userID, text)
	}

	h.logger.Info().Int64("user_id", userID).Int("media_count", len(mediaURLs)).Msg("Sending message with media")

	mediaInfos, err := h.validateAndClassifyMedia(mediaURLs)
	if err != nil {
		return fmt.Errorf("media validation failed: %w", err)
	}

	if len(mediaInfos) == 1 {
		return h.sendSingleMedia(ctx, userID, text, mediaInfos[0])
	} else if len(mediaInfos) <= MaxMediaGroupSize {
		return h.sendMediaGroup(ctx, userID, text, mediaInfos)
	}

	return h.sendMultipleMediaGroups(ctx, userID, text, mediaInfos)
}

// SendChatAction implements deps.TelegramSender interface
func (h *Handlers) SendChatAction(ctx context.Context, userID int64, action string) error {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.SendChatAction(msgCtx, &tgbot.SendChatActionParams{
		ChatID: userID,
		Action: models.ChatAction(action),
	})

	if err != nil {
		h.logger.Warn().Int64("user_id", userID).Str("action", action).Err(err).Msg("Failed to send chat action")
	}

	return err
}

// HandleStart handles /start command
func (h *Handlers) HandleStart(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(int64(userID), "/start", "processing")

	req := &dto.StartCommandRequest{
		UserID:   int64(userID),
		Username: update.Message.From.Username,
	}

	resp, err := h.uc.HandleStart(ctx, req)
	if err != nil {
		h.logError(int64(userID), "/start", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при обработке команды /start")
		return
	}

	h.sendResponse(ctx, chatID, resp.Message)
	h.logCommand(int64(userID), "/start", "success")
}

// HandleHelp handles /help command
func (h *Handlers) HandleHelp(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(int64(userID), "/help", "processing")

	resp, err := h.uc.HandleHelp(ctx)
	if err != nil {
		h.logError(int64(userID), "/help", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при обработке команды /help")
		return
	}

	h.sendResponse(ctx, chatID, resp.Message)
	h.logCommand(int64(userID), "/help", "success")
}

// HandleForwardedMessage handles forwarded messages from public channels (toggle subscription)
func (h *Handlers) HandleForwardedMessage(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	channelID, channelName, err := h.extractChannelFromForward(update.Message)
	if err != nil {
		h.logger.Warn().Err(err).Int64("user_id", int64(userID)).Msg("Failed to extract channel from forwarded message")
		h.sendResponse(ctx, chatID, "❌ Не удалось определить канал из пересланного сообщения")
		return
	}

	h.logCommand(int64(userID), "forward", fmt.Sprintf("channel: %s", channelID))

	req := &dto.ToggleSubscriptionRequest{
		UserID:      int64(userID),
		ChannelID:   channelID,
		ChannelName: channelName,
	}

	err = h.uc.HandleToggleSubscription(ctx, req)
	if err != nil {
		h.logError(int64(userID), "forward", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при обработке подписки")
		return
	}

	h.logCommand(int64(userID), "forward", "requested")
}

// extractChannelFromForward extracts channel ID and name from forwarded message
func (h *Handlers) extractChannelFromForward(msg *models.Message) (channelID, channelName string, err error) {
	if msg.ForwardOrigin == nil {
		return "", "", fmt.Errorf("message is not forwarded")
	}

	if msg.ForwardOrigin.Type != models.MessageOriginTypeChannel {
		return "", "", fmt.Errorf("message is not forwarded from channel")
	}

	channel := msg.ForwardOrigin.MessageOriginChannel
	if channel == nil {
		return "", "", fmt.Errorf("channel data is nil")
	}

	// Prefer @username, fallback to numeric ID for private channels
	if channel.Chat.Username != "" {
		channelID = "@" + channel.Chat.Username
		channelName = channel.Chat.Username
	} else {
		channelID = fmt.Sprintf("%d", channel.Chat.ID)
		channelName = channel.Chat.Title
		if channelName == "" {
			channelName = channelID
		}
	}

	return channelID, channelName, nil
}

// HandleList handles /list command
func (h *Handlers) HandleList(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.logCommand(int64(userID), "/list", "processing")

	resp, err := h.uc.HandleListSubscriptions(ctx, int64(userID))
	if err != nil {
		h.logError(int64(userID), "/list", err)
		h.sendResponse(ctx, chatID, "❌ Произошла ошибка при получении списка подписок")
		return
	}

	result := h.formatSubscriptions(resp)
	h.sendResponse(ctx, chatID, result)
	h.logCommand(int64(userID), "/list", "success")
}

func (h *Handlers) formatSubscriptions(resp *dto.SubscriptionListResponse) string {
	if len(resp.Subscriptions) == 0 {
		return "📋 У вас пока нет подписок"
	}

	var result strings.Builder
	result.WriteString("📋 <b>Ваши подписки:</b>\n")

	for _, sub := range resp.Subscriptions {
		result.WriteString(fmt.Sprintf("• <code>%s</code>\n", sub.ChannelName))
	}

	return result.String()
}

func (h *Handlers) sendResponse(ctx context.Context, chatID int64, text string) {
	if err := h.SendMessage(ctx, chatID, text); err != nil {
		h.logger.Error().Int64("chat_id", chatID).Err(err).Msg("Failed to send Telegram response")
	}
}

func (h *Handlers) sendSingleMessage(ctx context.Context, userID int64, text string) error {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.SendMessage(msgCtx, &tgbot.SendMessageParams{
		ChatID:    userID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})

	if err != nil {
		handledErr := h.handleSendMessageError(userID, err)
		h.logMessageSend(userID, len(text), false, handledErr)
		return handledErr
	}

	h.logMessageSend(userID, len(text), true, nil)
	return nil
}

func (h *Handlers) sendSplitMessage(ctx context.Context, userID int64, text string) error {
	h.logger.Info().Int64("user_id", userID).Int("total_length", len(text)).Msg("Splitting long message into parts")

	parts := h.splitMessage(text)
	totalParts := len(parts)
	successCount := 0

	for i, part := range parts {
		partNumber := i + 1

		if totalParts > 1 {
			part = fmt.Sprintf("<i>(Часть %d/%d)</i>\n\n%s", partNumber, totalParts, part)
		}

		err := h.sendSingleMessage(ctx, userID, part)
		if err != nil {
			h.logger.Error().Int64("user_id", userID).Int("part", partNumber).Int("total_parts", totalParts).Err(err).Msg("Failed to send message part")
			continue
		}

		successCount++

		if partNumber < totalParts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	h.logger.Info().Int64("user_id", userID).Int("success_parts", successCount).Int("total_parts", totalParts).Msg("Finished sending split message")

	if successCount == 0 {
		return fmt.Errorf("failed to send all message parts")
	}

	if successCount < totalParts {
		return fmt.Errorf("sent only %d out of %d message parts", successCount, totalParts)
	}

	return nil
}

func (h *Handlers) splitMessage(text string) []string {
	if len(text) <= MaxMessageLength {
		return []string{text}
	}

	var parts []string
	lines := strings.Split(text, "\n")
	currentPart := strings.Builder{}
	currentLength := 0

	for _, line := range lines {
		lineLength := len(line) + 1

		if currentLength+lineLength > MaxMessageLength {
			if currentPart.Len() > 0 {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
				currentLength = 0
			}

			if lineLength > MaxMessageLength {
				splitLines := h.splitLongLine(line)
				parts = append(parts, splitLines...)
				continue
			}
		}

		if currentPart.Len() > 0 {
			currentPart.WriteString("\n")
			currentLength++
		}
		currentPart.WriteString(line)
		currentLength += len(line)
	}

	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	return parts
}

func (h *Handlers) splitLongLine(line string) []string {
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

		if end < len(line) {
			lastSpace := strings.LastIndex(line[start:end], " ")
			if lastSpace > 0 {
				end = start + lastSpace
			}
		}

		parts = append(parts, line[start:end])
		start = end

		for start < len(line) && line[start] == ' ' {
			start++
		}
	}

	return parts
}

func (h *Handlers) handleSendMessageError(userID int64, err error) error {
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "Forbidden"):
		h.logger.Warn().Int64("user_id", userID).Msg("User blocked the bot or chat not found")
		return fmt.Errorf("user blocked the bot or chat not found")

	case strings.Contains(errorMsg, "Bad Request: chat not found"):
		h.logger.Warn().Int64("user_id", userID).Msg("Chat not found")
		return fmt.Errorf("chat not found")

	case strings.Contains(errorMsg, "Too Many Requests"):
		h.logger.Warn().Int64("user_id", userID).Msg("Rate limit exceeded")
		return fmt.Errorf("rate limit exceeded, please try again later")

	case strings.Contains(errorMsg, "network error"), strings.Contains(errorMsg, "timeout"):
		h.logger.Warn().Int64("user_id", userID).Msg("Network error while sending message")
		return fmt.Errorf("network error, please try again")

	default:
		h.logger.Error().Int64("user_id", userID).Err(err).Msg("Unknown error while sending message")
		return fmt.Errorf("failed to send message: %w", err)
	}
}

// logMessageSend logs message send result
func (h *Handlers) logMessageSend(userID int64, length int, success bool, err error) {
	logEvent := h.logger.Info()
	if !success {
		logEvent = h.logger.Error()
	}

	logEvent.Int64("user_id", userID).Int("message_length", length).Bool("success", success)

	if err != nil {
		logEvent.Err(err)
	}

	logEvent.Msg("Message send attempt completed")
}

func (h *Handlers) validateAndClassifyMedia(mediaURLs []string) ([]MediaInfo, error) {
	var mediaInfos []MediaInfo

	for _, mediaURL := range mediaURLs {
		if err := h.validateMediaURL(mediaURL); err != nil {
			return nil, err
		}

		mediaInfo, err := h.classifyMedia(mediaURL)
		if err != nil {
			return nil, err
		}

		if err := h.checkFileSize(mediaInfo); err != nil {
			return nil, err
		}

		mediaInfos = append(mediaInfos, mediaInfo)
	}

	return mediaInfos, nil
}

func (h *Handlers) validateMediaURL(mediaURL string) error {
	// Check if this is a Telegram file_id (no URL scheme)
	// file_id is a base64-encoded string without "://"
	if !strings.Contains(mediaURL, "://") {
		// This is a file_id, accept it without validation
		// Telegram will handle invalid file_ids gracefully
		return nil
	}

	// For URLs, do basic validation
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return fmt.Errorf("invalid URL format '%s': %w", mediaURL, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme '%s' for '%s'", parsedURL.Scheme, mediaURL)
	}

	// Skip HEAD request - it can fail for many legitimate URLs
	// and Telegram will handle invalid URLs gracefully
	return nil
}

// classifyMedia determines media file type by URL or file_id
func (h *Handlers) classifyMedia(mediaURL string) (MediaInfo, error) {
	// Check if this is a Telegram file_id (no URL scheme)
	if !strings.Contains(mediaURL, "://") {
		// This is a file_id - use Photo as default type
		// Telegram Bot API will correctly handle the file_id regardless of the method used
		// sendPhoto, sendVideo, sendDocument all accept file_id and Telegram determines the actual type
		return MediaInfo{
			URL:      mediaURL,
			Type:     MediaTypePhoto, // Default to photo, works for most file_ids
			MimeType: "",
			FileName: "",
		}, nil
	}

	// For URLs, determine type from extension and content-type
	parsedURL, _ := url.Parse(mediaURL)
	fileName := path.Base(parsedURL.Path)
	ext := strings.ToLower(path.Ext(fileName))

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
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

	mediaType := h.determineMediaType(mimeType, ext)

	return MediaInfo{
		URL:      mediaURL,
		Type:     mediaType,
		MimeType: mimeType,
		FileName: fileName,
	}, nil
}

func (h *Handlers) determineMediaType(mimeType, ext string) MediaType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return MediaTypePhoto
	case strings.HasPrefix(mimeType, "video/"):
		return MediaTypeVideo
	case strings.HasPrefix(mimeType, "application/") || strings.HasPrefix(mimeType, "text/"):
		return MediaTypeDocument
	}

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

func (h *Handlers) checkFileSize(mediaInfo MediaInfo) error {
	// Skip size check for file_id - Telegram already has the file, no need to check
	if !strings.Contains(mediaInfo.URL, "://") {
		return nil
	}

	req, err := http.NewRequest("HEAD", mediaInfo.URL, nil)
	if err != nil {
		// Don't fail on HEAD request error, Telegram will handle it
		h.logger.Warn().Str("url", mediaInfo.URL).Err(err).Msg("Could not create HEAD request, skipping size check")
		return nil
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		// Don't fail on network error, let Telegram try to fetch
		h.logger.Warn().Str("url", mediaInfo.URL).Err(err).Msg("Could not check file size, proceeding anyway")
		return nil
	}
	defer resp.Body.Close()

	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		h.logger.Warn().Str("url", mediaInfo.URL).Msg("Could not determine file size, proceeding anyway")
		return nil
	}

	var fileSize int64
	fmt.Sscanf(contentLength, "%d", &fileSize)

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

func (h *Handlers) sendSingleMedia(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
	h.logger.Debug().Int64("user_id", userID).Str("media_type", string(mediaInfo.Type)).Str("url", mediaInfo.URL).Msg("Sending single media")

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

		h.logger.Warn().Int64("user_id", userID).Int("attempt", attempt).Err(err).Msg("Failed to send media, retrying")

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

func (h *Handlers) sendPhoto(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
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

func (h *Handlers) sendVideo(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
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

func (h *Handlers) sendDocument(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) error {
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

func (h *Handlers) sendMediaGroup(ctx context.Context, userID int64, text string, mediaInfos []MediaInfo) error {
	h.logger.Debug().Int64("user_id", userID).Int("media_count", len(mediaInfos)).Msg("Sending media group")

	if len(mediaInfos) == 1 {
		return h.sendSingleMedia(ctx, userID, text, mediaInfos[0])
	}

	if err := h.sendSingleMedia(ctx, userID, text, mediaInfos[0]); err != nil {
		return fmt.Errorf("failed to send first media: %w", err)
	}

	for i := 1; i < len(mediaInfos); i++ {
		if err := h.sendSingleMedia(ctx, userID, "", mediaInfos[i]); err != nil {
			h.logger.Error().Int64("user_id", userID).Int("media_index", i).Err(err).Msg("Failed to send media in group")
		}

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

func (h *Handlers) sendMultipleMediaGroups(ctx context.Context, userID int64, text string, mediaInfos []MediaInfo) error {
	h.logger.Info().Int64("user_id", userID).Int("total_media", len(mediaInfos)).Msg("Sending multiple media groups")

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

	if err := h.sendMediaGroup(ctx, userID, text, groups[0]); err != nil {
		h.logger.Error().Int64("user_id", userID).Int("group", 1).Err(err).Msg("Failed to send first media group")
	} else {
		successCount++
	}

	for i := 1; i < totalGroups; i++ {
		groupText := ""
		if totalGroups > 1 {
			groupText = fmt.Sprintf("<i>(Медиа %d/%d)</i>", i+1, totalGroups)
		}

		if err := h.sendMediaGroup(ctx, userID, groupText, groups[i]); err != nil {
			h.logger.Error().Int64("user_id", userID).Int("group", i+1).Err(err).Msg("Failed to send media group")
		} else {
			successCount++
		}

		if i < totalGroups-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	h.logger.Info().Int64("user_id", userID).Int("success_groups", successCount).Int("total_groups", totalGroups).Msg("Finished sending multiple media groups")

	if successCount == 0 {
		return fmt.Errorf("failed to send all media groups")
	}

	if successCount < totalGroups {
		return fmt.Errorf("sent only %d out of %d media groups", successCount, totalGroups)
	}

	return nil
}

// logMediaSend logs media send result
func (h *Handlers) logMediaSend(userID int64, mediaCount int, success bool, err error) {
	logEvent := h.logger.Info()
	if !success {
		logEvent = h.logger.Error()
	}

	logEvent.Int64("user_id", userID).Int("media_count", mediaCount).Bool("success", success)

	if err != nil {
		logEvent.Err(err)
	}

	logEvent.Msg("Media send attempt completed")
}

// logCommand logs successful commands
func (h *Handlers) logCommand(userID int64, command, result string) {
	h.logger.Info().Int64("user_id", userID).Str("command", command).Str("result", result).Msg("Telegram command processed")
}

// logError logs command errors
func (h *Handlers) logError(userID int64, command string, err error) {
	h.logger.Error().Int64("user_id", userID).Str("command", command).Err(err).Msg("Telegram command failed")
}

// SendMessageAndGetID sends a text message and returns the telegram message ID
func (h *Handlers) SendMessageAndGetID(ctx context.Context, userID int64, text string) (int, error) {
	if text == "" {
		h.logger.Warn().Int64("user_id", userID).Msg("Attempt to send empty message")
		return 0, fmt.Errorf("message text cannot be empty")
	}

	h.logger.Debug().Int64("user_id", userID).Int("text_length", len(text)).Msg("Sending message to user and getting ID")

	// For long messages, split and return last message ID
	if len(text) > MaxMessageLength {
		return h.sendSplitMessageAndGetID(ctx, userID, text)
	}

	return h.sendSingleMessageAndGetID(ctx, userID, text)
}

func (h *Handlers) sendSingleMessageAndGetID(ctx context.Context, userID int64, text string) (int, error) {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	msg, err := h.bot.SendMessage(msgCtx, &tgbot.SendMessageParams{
		ChatID:    userID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})

	if err != nil {
		handledErr := h.handleSendMessageError(userID, err)
		h.logMessageSend(userID, len(text), false, handledErr)
		return 0, handledErr
	}

	h.logMessageSend(userID, len(text), true, nil)
	return msg.ID, nil
}

func (h *Handlers) sendSplitMessageAndGetID(ctx context.Context, userID int64, text string) (int, error) {
	h.logger.Info().Int64("user_id", userID).Int("total_length", len(text)).Msg("Splitting long message into parts")

	parts := h.splitMessage(text)
	totalParts := len(parts)
	var lastMessageID int

	for i, part := range parts {
		partNumber := i + 1

		if totalParts > 1 {
			part = fmt.Sprintf("<i>(Часть %d/%d)</i>\n\n%s", partNumber, totalParts, part)
		}

		msgID, err := h.sendSingleMessageAndGetID(ctx, userID, part)
		if err != nil {
			h.logger.Error().Int64("user_id", userID).Int("part", partNumber).Int("total_parts", totalParts).Err(err).Msg("Failed to send message part")
			if lastMessageID == 0 {
				return 0, err
			}
			return lastMessageID, nil
		}

		lastMessageID = msgID

		if partNumber < totalParts {
			select {
			case <-ctx.Done():
				return lastMessageID, ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	h.logger.Info().Int64("user_id", userID).Int("total_parts", totalParts).Int("last_message_id", lastMessageID).Msg("Finished sending split message")
	return lastMessageID, nil
}

// SendMessageWithMediaAndGetID sends a message with media and returns the telegram message ID
func (h *Handlers) SendMessageWithMediaAndGetID(ctx context.Context, userID int64, text string, mediaURLs []string) (int, error) {
	if len(mediaURLs) == 0 {
		return h.SendMessageAndGetID(ctx, userID, text)
	}

	h.logger.Info().Int64("user_id", userID).Int("media_count", len(mediaURLs)).Msg("Sending message with media and getting ID")

	mediaInfos, err := h.validateAndClassifyMedia(mediaURLs)
	if err != nil {
		return 0, fmt.Errorf("media validation failed: %w", err)
	}

	if len(mediaInfos) == 1 {
		return h.sendSingleMediaAndGetID(ctx, userID, text, mediaInfos[0])
	}

	// For multiple media, send first one with caption and return its ID
	msgID, err := h.sendSingleMediaAndGetID(ctx, userID, text, mediaInfos[0])
	if err != nil {
		return 0, err
	}

	// Send remaining media without caption
	for i := 1; i < len(mediaInfos); i++ {
		if _, sendErr := h.sendSingleMediaAndGetID(ctx, userID, "", mediaInfos[i]); sendErr != nil {
			h.logger.Error().Int64("user_id", userID).Int("media_index", i).Err(sendErr).Msg("Failed to send additional media")
		}

		if i < len(mediaInfos)-1 {
			select {
			case <-ctx.Done():
				return msgID, ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	return msgID, nil
}

func (h *Handlers) sendSingleMediaAndGetID(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) (int, error) {
	h.logger.Debug().Int64("user_id", userID).Str("media_type", string(mediaInfo.Type)).Str("url", mediaInfo.URL).Msg("Sending single media and getting ID")

	var msg *models.Message
	var err error

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		switch mediaInfo.Type {
		case MediaTypePhoto:
			msg, err = h.sendPhotoAndGetID(ctx, userID, text, mediaInfo)
		case MediaTypeVideo:
			msg, err = h.sendVideoAndGetID(ctx, userID, text, mediaInfo)
		case MediaTypeDocument:
			msg, err = h.sendDocumentAndGetID(ctx, userID, text, mediaInfo)
		default:
			return 0, fmt.Errorf("unsupported media type: %s", mediaInfo.Type)
		}

		if err == nil {
			break
		}

		h.logger.Warn().Int64("user_id", userID).Int("attempt", attempt).Err(err).Msg("Failed to send media, retrying")

		if attempt < MaxRetries {
			time.Sleep(RetryDelay * time.Duration(attempt))
		}
	}

	if err != nil {
		h.logMediaSend(userID, 1, false, err)
		return 0, fmt.Errorf("failed to send media after %d attempts: %w", MaxRetries, err)
	}

	h.logMediaSend(userID, 1, true, nil)
	return msg.ID, nil
}

func (h *Handlers) sendPhotoAndGetID(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) (*models.Message, error) {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	return h.bot.SendPhoto(msgCtx, &tgbot.SendPhotoParams{
		ChatID:    userID,
		Photo:     &models.InputFileString{Data: mediaInfo.URL},
		Caption:   text,
		ParseMode: models.ParseModeHTML,
	})
}

func (h *Handlers) sendVideoAndGetID(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) (*models.Message, error) {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	return h.bot.SendVideo(msgCtx, &tgbot.SendVideoParams{
		ChatID:    userID,
		Video:     &models.InputFileString{Data: mediaInfo.URL},
		Caption:   text,
		ParseMode: models.ParseModeHTML,
	})
}

func (h *Handlers) sendDocumentAndGetID(ctx context.Context, userID int64, text string, mediaInfo MediaInfo) (*models.Message, error) {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	return h.bot.SendDocument(msgCtx, &tgbot.SendDocumentParams{
		ChatID:    userID,
		Document:  &models.InputFileString{Data: mediaInfo.URL},
		Caption:   text,
		ParseMode: models.ParseModeHTML,
	})
}

// DeleteMessage deletes a message from user's chat
func (h *Handlers) DeleteMessage(ctx context.Context, userID int64, messageID int) error {
	h.logger.Debug().Int64("user_id", userID).Int("message_id", messageID).Msg("Deleting message")

	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	_, err := h.bot.DeleteMessage(msgCtx, &tgbot.DeleteMessageParams{
		ChatID:    userID,
		MessageID: messageID,
	})

	if err != nil {
		h.logger.Error().Int64("user_id", userID).Int("message_id", messageID).Err(err).Msg("Failed to delete message")
		return fmt.Errorf("failed to delete message: %w", err)
	}

	h.logger.Info().Int64("user_id", userID).Int("message_id", messageID).Msg("Message deleted successfully")
	return nil
}

// EditMessageText edits message text in user's chat
func (h *Handlers) EditMessageText(ctx context.Context, userID int64, messageID int, text string) error {
	if text == "" {
		return fmt.Errorf("message text cannot be empty")
	}

	h.logger.Debug().Int64("user_id", userID).Int("message_id", messageID).Int("text_length", len(text)).Msg("Editing message text")

	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	// Truncate text if too long
	if len(text) > MaxMessageLength {
		text = text[:MaxMessageLength-3] + "..."
	}

	_, err := h.bot.EditMessageText(msgCtx, &tgbot.EditMessageTextParams{
		ChatID:    userID,
		MessageID: messageID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})

	if err != nil {
		h.logger.Error().Int64("user_id", userID).Int("message_id", messageID).Err(err).Msg("Failed to edit message text")
		return fmt.Errorf("failed to edit message: %w", err)
	}

	h.logger.Info().Int64("user_id", userID).Int("message_id", messageID).Msg("Message edited successfully")
	return nil
}

// CopyMessageAndGetID copies a message from a public channel to user's chat
// Uses Bot API copyMessage method which handles media automatically
func (h *Handlers) CopyMessageAndGetID(ctx context.Context, userID int64, fromChannelID string, messageID int, caption string) (int, error) {
	h.logger.Info().
		Int64("user_id", userID).
		Str("from_channel", fromChannelID).
		Int("message_id", messageID).
		Msg("Copying message from channel")

	var copiedMsgID int
	var err error

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)

		result, copyErr := h.bot.CopyMessage(msgCtx, &tgbot.CopyMessageParams{
			ChatID:     userID,
			FromChatID: fromChannelID,
			MessageID:  messageID,
			Caption:    caption,
			ParseMode:  models.ParseModeHTML,
		})
		cancel()

		if copyErr == nil && result != nil {
			copiedMsgID = result.ID
			break
		}

		err = copyErr
		h.logger.Warn().
			Int64("user_id", userID).
			Str("from_channel", fromChannelID).
			Int("message_id", messageID).
			Int("attempt", attempt).
			Err(err).
			Msg("Failed to copy message, retrying")

		if attempt < MaxRetries {
			time.Sleep(RetryDelay * time.Duration(attempt))
		}
	}

	if err != nil {
		h.logger.Error().
			Int64("user_id", userID).
			Str("from_channel", fromChannelID).
			Int("message_id", messageID).
			Err(err).
			Msg("Failed to copy message after retries")
		return 0, fmt.Errorf("failed to copy message after %d attempts: %w", MaxRetries, err)
	}

	h.logger.Info().
		Int64("user_id", userID).
		Str("from_channel", fromChannelID).
		Int("message_id", messageID).
		Int("copied_message_id", copiedMsgID).
		Msg("Message copied successfully")

	return copiedMsgID, nil
}

// DownloadFiles downloads multiple files from S3 URLs
func (h *Handlers) DownloadFiles(ctx context.Context, urls []string) ([]*entities.DownloadedFile, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	h.logger.Info().Int("url_count", len(urls)).Msg("Downloading files from S3")

	files := make([]*entities.DownloadedFile, 0, len(urls))

	for _, fileURL := range urls {
		file, err := h.downloadFile(ctx, fileURL)
		if err != nil {
			h.logger.Warn().Err(err).Str("url", fileURL).Msg("Failed to download file, skipping")
			continue
		}
		files = append(files, file)
	}

	h.logger.Info().Int("downloaded_count", len(files)).Int("total_count", len(urls)).Msg("Files download completed")

	return files, nil
}

func (h *Handlers) downloadFile(ctx context.Context, fileURL string) (*entities.DownloadedFile, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, DownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(downloadCtx, "GET", fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("S3 returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	filename := extractFilename(fileURL)
	contentType := resp.Header.Get("Content-Type")
	mediaType := classifyMediaTypeFromContentType(contentType, filename)

	h.logger.Debug().
		Str("url", fileURL).
		Str("filename", filename).
		Str("content_type", contentType).
		Str("media_type", mediaType).
		Int("size_bytes", len(data)).
		Msg("File downloaded successfully")

	return &entities.DownloadedFile{
		Data:        data,
		Filename:    filename,
		ContentType: contentType,
		MediaType:   mediaType,
	}, nil
}

// extractFilename extracts filename from URL
func extractFilename(fileURL string) string {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "file"
	}
	filename := path.Base(parsedURL.Path)
	if filename == "" || filename == "." || filename == "/" {
		return "file"
	}
	return filename
}

// classifyMediaTypeFromContentType determines media type from content-type header
func classifyMediaTypeFromContentType(contentType, filename string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "photo"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	}

	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return "photo"
	case ".mp4", ".avi", ".mov", ".mkv", ".webm":
		return "video"
	default:
		return "document"
	}
}

// SendMessageWithFilesAndGetID sends downloaded files to user and returns telegram message ID
func (h *Handlers) SendMessageWithFilesAndGetID(ctx context.Context, userID int64, text string, files []*entities.DownloadedFile) (int, error) {
	if len(files) == 0 {
		return h.SendMessageAndGetID(ctx, userID, text)
	}

	h.logger.Info().Int64("user_id", userID).Int("file_count", len(files)).Msg("Sending message with downloaded files")

	// Send first file with caption
	msgID, err := h.sendFileAndGetID(ctx, userID, text, files[0])
	if err != nil {
		return 0, err
	}

	// Send remaining files without caption
	for i := 1; i < len(files); i++ {
		if _, sendErr := h.sendFileAndGetID(ctx, userID, "", files[i]); sendErr != nil {
			h.logger.Error().Err(sendErr).Int64("user_id", userID).Int("file_index", i).Msg("Failed to send additional file")
		}

		if i < len(files)-1 {
			select {
			case <-ctx.Done():
				return msgID, ctx.Err()
			case <-time.After(MessageSplitTimeout):
			}
		}
	}

	return msgID, nil
}

func (h *Handlers) sendFileAndGetID(ctx context.Context, userID int64, text string, file *entities.DownloadedFile) (int, error) {
	msgCtx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	var msg *models.Message
	var err error

	// Check if we already have file_id from previous upload
	if file.FileID != "" {
		// Use cached file_id - much faster, no upload needed
		msg, err = h.sendFileByID(msgCtx, userID, text, file)
	} else {
		// First time - upload file and get file_id
		msg, err = h.sendFileByUpload(msgCtx, userID, text, file)
	}

	if err != nil {
		h.logger.Error().Err(err).
			Int64("user_id", userID).
			Str("media_type", file.MediaType).
			Str("filename", file.Filename).
			Bool("used_file_id", file.FileID != "").
			Msg("Failed to send file")
		return 0, fmt.Errorf("failed to send %s: %w", file.MediaType, err)
	}

	// Extract file_id from response for reuse (only if we uploaded)
	if file.FileID == "" {
		file.FileID = h.extractFileID(msg, file.MediaType)
		if file.FileID != "" {
			// Clear raw data to free memory - we have file_id now
			file.Data = nil
			h.logger.Debug().
				Str("file_id", file.FileID).
				Str("media_type", file.MediaType).
				Msg("Extracted file_id, cleared raw data from memory")
		}
	}

	h.logger.Debug().
		Int64("user_id", userID).
		Int("message_id", msg.ID).
		Str("media_type", file.MediaType).
		Str("filename", file.Filename).
		Bool("used_file_id", file.FileID != "").
		Msg("File sent successfully")

	return msg.ID, nil
}

// sendFileByUpload uploads file data to Telegram
func (h *Handlers) sendFileByUpload(ctx context.Context, userID int64, text string, file *entities.DownloadedFile) (*models.Message, error) {
	switch file.MediaType {
	case "photo":
		return h.bot.SendPhoto(ctx, &tgbot.SendPhotoParams{
			ChatID:    userID,
			Photo:     &models.InputFileUpload{Filename: file.Filename, Data: bytes.NewReader(file.Data)},
			Caption:   text,
			ParseMode: models.ParseModeHTML,
		})
	case "video":
		return h.bot.SendVideo(ctx, &tgbot.SendVideoParams{
			ChatID:    userID,
			Video:     &models.InputFileUpload{Filename: file.Filename, Data: bytes.NewReader(file.Data)},
			Caption:   text,
			ParseMode: models.ParseModeHTML,
		})
	default:
		return h.bot.SendDocument(ctx, &tgbot.SendDocumentParams{
			ChatID:    userID,
			Document:  &models.InputFileUpload{Filename: file.Filename, Data: bytes.NewReader(file.Data)},
			Caption:   text,
			ParseMode: models.ParseModeHTML,
		})
	}
}

// sendFileByID sends file using cached Telegram file_id
func (h *Handlers) sendFileByID(ctx context.Context, userID int64, text string, file *entities.DownloadedFile) (*models.Message, error) {
	switch file.MediaType {
	case "photo":
		return h.bot.SendPhoto(ctx, &tgbot.SendPhotoParams{
			ChatID:    userID,
			Photo:     &models.InputFileString{Data: file.FileID},
			Caption:   text,
			ParseMode: models.ParseModeHTML,
		})
	case "video":
		return h.bot.SendVideo(ctx, &tgbot.SendVideoParams{
			ChatID:    userID,
			Video:     &models.InputFileString{Data: file.FileID},
			Caption:   text,
			ParseMode: models.ParseModeHTML,
		})
	default:
		return h.bot.SendDocument(ctx, &tgbot.SendDocumentParams{
			ChatID:    userID,
			Document:  &models.InputFileString{Data: file.FileID},
			Caption:   text,
			ParseMode: models.ParseModeHTML,
		})
	}
}

// extractFileID extracts file_id from Telegram response
func (h *Handlers) extractFileID(msg *models.Message, mediaType string) string {
	if msg == nil {
		return ""
	}

	switch mediaType {
	case "photo":
		// Photo array contains multiple sizes, take the largest (last one)
		if len(msg.Photo) > 0 {
			return msg.Photo[len(msg.Photo)-1].FileID
		}
	case "video":
		if msg.Video != nil {
			return msg.Video.FileID
		}
	default:
		if msg.Document != nil {
			return msg.Document.FileID
		}
	}

	return ""
}
