package handlers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	channeldeps "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/metrics"
)

const (
	// AlbumBufferTimeout is the time to wait for all album parts to arrive
	AlbumBufferTimeout = 500 * time.Millisecond
)

// pendingAlbum represents an album being collected
type pendingAlbum struct {
	items []domain.NewsItem
	timer *time.Timer
	mu    sync.Mutex
}

// NewsUpdateHandler handles real-time news updates from Telegram
type NewsUpdateHandler struct {
	channelRepo channeldeps.ChannelRepository
	msgIDCache  channeldeps.MessageIDCache
	producer    domain.KafkaProducer
	logger      zerolog.Logger
	metrics     *metrics.Metrics

	// Media processing
	mediaUploader    domain.MediaUploader
	mediaDownloadFn  domain.MediaDownloadFunc
	mediaDownloadMu  sync.RWMutex

	// Album buffering for grouping media
	albumBuffer map[int64]*pendingAlbum
	albumMu     sync.Mutex

	// Health tracking
	healthy        atomic.Bool
	lastUpdateTime atomic.Value
	processedCount atomic.Int64
}

// NewNewsUpdateHandler creates a new handler for processing Telegram channel updates
func NewNewsUpdateHandler(
	channelRepo channeldeps.ChannelRepository,
	msgIDCache channeldeps.MessageIDCache,
	producer domain.KafkaProducer,
	logger zerolog.Logger,
	m *metrics.Metrics,
	mediaUploader domain.MediaUploader,
) *NewsUpdateHandler {
	h := &NewsUpdateHandler{
		channelRepo:   channelRepo,
		msgIDCache:    msgIDCache,
		producer:      producer,
		logger:        logger.With().Str("component", "news_update_handler").Logger(),
		metrics:       m,
		mediaUploader: mediaUploader,
		albumBuffer:   make(map[int64]*pendingAlbum),
	}
	h.healthy.Store(true)
	h.lastUpdateTime.Store(time.Now())
	return h
}

// SetMediaDownloadFunc sets the function for downloading media from Telegram
// This is called by MTProtoClient after connection to provide access to tg.Client
func (h *NewsUpdateHandler) SetMediaDownloadFunc(fn domain.MediaDownloadFunc) {
	h.mediaDownloadMu.Lock()
	defer h.mediaDownloadMu.Unlock()
	h.mediaDownloadFn = fn
	h.logger.Info().Msg("media download function set")
}

// getMediaDownloadFunc safely retrieves the media download function
func (h *NewsUpdateHandler) getMediaDownloadFunc() domain.MediaDownloadFunc {
	h.mediaDownloadMu.RLock()
	defer h.mediaDownloadMu.RUnlock()
	return h.mediaDownloadFn
}

// OnNewChannelMessage handles new channel message updates
// This is the callback for tg.UpdateDispatcher.OnNewChannelMessage
func (h *NewsUpdateHandler) OnNewChannelMessage(
	ctx context.Context,
	e tg.Entities,
	u *tg.UpdateNewChannelMessage,
) error {
	h.lastUpdateTime.Store(time.Now())

	msg, ok := u.Message.(*tg.Message)
	if !ok {
		h.logger.Debug().Msg("skipping non-message update")
		return nil
	}

	if msg.Out {
		h.logger.Debug().Int("message_id", msg.ID).Msg("skipping outgoing message")
		return nil
	}

	channelID, channelName := h.extractChannelInfo(msg, e)
	if channelID == "" {
		h.logger.Debug().Int("message_id", msg.ID).Msg("could not extract channel ID from message")
		return nil
	}

	exists, err := h.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to check if channel exists")
		return nil
	}

	if !exists {
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", msg.ID).
			Msg("received message from unsubscribed channel, ignoring")
		return nil
	}

	// Check message ID against in-memory cache first (prevents race condition with fallback collector)
	if lastProcessedID, found := h.msgIDCache.Get(channelID); found {
		if msg.ID <= lastProcessedID {
			h.logger.Debug().
				Str("channel_id", channelID).
				Int("message_id", msg.ID).
				Int("last_processed", lastProcessedID).
				Msg("skipping already processed message (from cache)")
			return nil
		}
	}

	channel, err := h.channelRepo.GetChannel(ctx, channelID)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to get channel")
		return nil
	}

	// Double-check against DB value (cache might not have this channel yet)
	if msg.ID <= channel.LastProcessedMessageID {
		// Update cache with DB value for future checks
		h.msgIDCache.SetIfGreater(channelID, channel.LastProcessedMessageID)
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", msg.ID).
			Int("last_processed", channel.LastProcessedMessageID).
			Msg("skipping already processed message (from DB)")
		return nil
	}

	if channelName == "" && channel.ChannelName != "" {
		channelName = channel.ChannelName
	}

	newsItem := h.convertToNewsItem(ctx, msg, channelID, channelName)

	// Handle album grouping
	if newsItem.GroupedID != 0 {
		h.addToAlbumBuffer(ctx, newsItem, channelID, msg.ID)
	} else {
		// Non-album message, send immediately
		h.sendNewsItem(ctx, &newsItem, channelID, msg.ID)
	}

	return nil
}

// addToAlbumBuffer adds an item to the album buffer for grouping
func (h *NewsUpdateHandler) addToAlbumBuffer(ctx context.Context, item domain.NewsItem, channelID string, msgID int) {
	h.albumMu.Lock()

	album, exists := h.albumBuffer[item.GroupedID]
	if !exists {
		// Create new pending album
		album = &pendingAlbum{
			items: make([]domain.NewsItem, 0, 10),
		}
		h.albumBuffer[item.GroupedID] = album

		// Start timer for flushing
		groupedID := item.GroupedID
		album.timer = time.AfterFunc(AlbumBufferTimeout, func() {
			h.flushAlbum(context.Background(), groupedID)
		})

		h.logger.Debug().
			Int64("grouped_id", item.GroupedID).
			Str("channel_id", channelID).
			Msg("created new pending album")
	}

	album.mu.Lock()
	album.items = append(album.items, item)
	album.mu.Unlock()

	h.logger.Debug().
		Int64("grouped_id", item.GroupedID).
		Int("message_id", msgID).
		Int("album_size", len(album.items)).
		Msg("added item to album buffer")

	h.albumMu.Unlock()
}

// flushAlbum combines all items in an album and sends the result
func (h *NewsUpdateHandler) flushAlbum(ctx context.Context, groupedID int64) {
	h.albumMu.Lock()
	album, exists := h.albumBuffer[groupedID]
	if !exists {
		h.albumMu.Unlock()
		return
	}
	delete(h.albumBuffer, groupedID)
	h.albumMu.Unlock()

	album.mu.Lock()
	items := album.items
	album.mu.Unlock()

	if len(items) == 0 {
		return
	}

	// Combine album items into a single NewsItem
	combined := h.combineAlbumItems(items)

	h.logger.Info().
		Int64("grouped_id", groupedID).
		Str("channel_id", combined.ChannelID).
		Int("items_count", len(items)).
		Int("media_count", len(combined.MediaURLs)).
		Msg("flushing album")

	h.sendNewsItem(ctx, &combined, combined.ChannelID, combined.MessageID)
}

// combineAlbumItems merges multiple album items into a single NewsItem
func (h *NewsUpdateHandler) combineAlbumItems(items []domain.NewsItem) domain.NewsItem {
	if len(items) == 0 {
		return domain.NewsItem{}
	}

	// Use the first item as base
	result := items[0]

	// Collect all media URLs, metadata and find text content
	var allMediaURLs []string
	var allMetadata []domain.MediaMetadata
	var textContent string

	for _, item := range items {
		allMediaURLs = append(allMediaURLs, item.MediaURLs...)
		allMetadata = append(allMetadata, item.MediaMetadata...)
		if item.Content != "" && textContent == "" {
			textContent = item.Content // Use first non-empty content
		}
	}

	result.MediaURLs = allMediaURLs
	result.MediaMetadata = allMetadata
	result.Content = textContent

	// Use the lowest message ID (first message in album)
	for _, item := range items {
		if item.MessageID < result.MessageID {
			result.MessageID = item.MessageID
		}
		if item.Date.Before(result.Date) {
			result.Date = item.Date
		}
	}

	return result
}

// sendNewsItem sends a news item to Kafka and updates tracking
func (h *NewsUpdateHandler) sendNewsItem(ctx context.Context, newsItem *domain.NewsItem, channelID string, msgID int) {
	// CRITICAL: Update cache FIRST (instant, prevents race condition with fallback collector)
	if !h.msgIDCache.SetIfGreater(channelID, msgID) {
		// Another goroutine already processed a higher message ID
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", msgID).
			Msg("skipping message - higher ID already in cache")
		return
	}

	if err := h.producer.SendNewsReceived(ctx, newsItem); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", msgID).
			Msg("failed to send news to Kafka")
		h.metrics.RecordKafkaError("send_failed")
		return
	}

	// Update DB asynchronously (persistence for restarts)
	if err := h.channelRepo.UpdateLastProcessedMessageID(ctx, channelID, msgID); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", msgID).
			Msg("failed to update last processed message ID in DB")
	}

	h.processedCount.Add(1)
	h.metrics.RecordKafkaMessage(0)

	h.logger.Info().
		Str("channel_id", channelID).
		Str("channel_name", newsItem.ChannelName).
		Int("message_id", msgID).
		Int("content_length", len(newsItem.Content)).
		Int("media_count", len(newsItem.MediaURLs)).
		Msg("real-time news sent to Kafka")
}

// FlushAlbumBuffer flushes all pending albums (for graceful shutdown)
func (h *NewsUpdateHandler) FlushAlbumBuffer() {
	h.albumMu.Lock()
	groupedIDs := make([]int64, 0, len(h.albumBuffer))
	for id, album := range h.albumBuffer {
		groupedIDs = append(groupedIDs, id)
		if album.timer != nil {
			album.timer.Stop()
		}
	}
	h.albumMu.Unlock()

	for _, id := range groupedIDs {
		h.flushAlbum(context.Background(), id)
	}

	h.logger.Info().Int("albums_flushed", len(groupedIDs)).Msg("flushed all pending albums")
}

// extractChannelInfo extracts channel ID and name from message and entities
func (h *NewsUpdateHandler) extractChannelInfo(msg *tg.Message, e tg.Entities) (string, string) {
	peerID := msg.GetPeerID()
	if peerID == nil {
		return "", ""
	}

	peerChannel, ok := peerID.(*tg.PeerChannel)
	if !ok {
		return "", ""
	}

	channelID := fmt.Sprintf("%d", peerChannel.ChannelID)
	channelName := ""

	if channel, ok := e.Channels[peerChannel.ChannelID]; ok {
		if channel.Title != "" {
			channelName = channel.Title
		}
		if channel.Username != "" {
			channelID = "@" + channel.Username
		}
	}

	return channelID, channelName
}

// convertToNewsItem converts a Telegram message to domain.NewsItem
func (h *NewsUpdateHandler) convertToNewsItem(ctx context.Context, msg *tg.Message, channelID, channelName string) domain.NewsItem {
	urls, metadata := h.extractMediaWithMetadata(ctx, msg.Media, channelID, msg.ID)

	newsItem := domain.NewsItem{
		ChannelID:     channelID,
		ChannelName:   channelName,
		MessageID:     msg.ID,
		Content:       msg.Message,
		MediaURLs:     urls,
		MediaMetadata: metadata,
		Date:          time.Unix(int64(msg.Date), 0),
		GroupedID:     msg.GroupedID, // For album grouping
	}

	return newsItem
}

// extractMediaWithMetadata extracts media from message and uploads to S3
// Downloads media via MTProto, uploads to MinIO S3, returns HTTP URLs and metadata
// Falls back to skipping media if download/upload fails
func (h *NewsUpdateHandler) extractMediaWithMetadata(ctx context.Context, media tg.MessageMediaClass, channelID string, messageID int) ([]string, []domain.MediaMetadata) {
	if media == nil {
		return []string{}, []domain.MediaMetadata{}
	}

	var urls []string
	var metadata []domain.MediaMetadata

	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		// Download photo and upload to S3
		url := h.processMediaToS3(ctx, media, channelID, messageID, "photo")
		if url != "" {
			urls = append(urls, url)
			metadata = append(metadata, domain.MediaMetadata{Type: domain.MediaTypePhoto})
		}

	case *tg.MessageMediaDocument:
		// Extract metadata from document attributes
		var meta domain.MediaMetadata
		if doc, ok := m.Document.(*tg.Document); ok {
			meta = h.extractDocumentMetadata(doc)
		} else {
			meta = domain.MediaMetadata{Type: domain.MediaTypeDocument}
		}

		// Download document and upload to S3
		url := h.processMediaToS3(ctx, media, channelID, messageID, meta.Type)
		if url != "" {
			urls = append(urls, url)
			metadata = append(metadata, meta)
		}

	case *tg.MessageMediaWebPage:
		if webpage, ok := m.Webpage.(*tg.WebPage); ok {
			// Keep webpage URL as-is (it's already a HTTP URL)
			if webpage.URL != "" {
				urls = append(urls, webpage.URL)
				metadata = append(metadata, domain.MediaMetadata{Type: domain.MediaTypeDocument})
			}
			// Also process photo from webpage if present
			if webpage.Photo != nil {
				url := h.processMediaToS3(ctx, &tg.MessageMediaPhoto{Photo: webpage.Photo}, channelID, messageID, "webpage_photo")
				if url != "" {
					urls = append(urls, url)
					metadata = append(metadata, domain.MediaMetadata{Type: domain.MediaTypePhoto})
				}
			}
		}

	case *tg.MessageMediaGeo:
		if m.Geo != nil {
			if geoPoint, ok := m.Geo.(*tg.GeoPoint); ok {
				geoURL := fmt.Sprintf("geo:%.6f,%.6f", geoPoint.Lat, geoPoint.Long)
				urls = append(urls, geoURL)
				metadata = append(metadata, domain.MediaMetadata{Type: domain.MediaTypeDocument})
			}
		}

	case *tg.MessageMediaContact:
		contactURL := fmt.Sprintf("contact://tel:%s", m.PhoneNumber)
		urls = append(urls, contactURL)
		metadata = append(metadata, domain.MediaMetadata{Type: domain.MediaTypeDocument})
	}

	return urls, metadata
}

// extractDocumentMetadata extracts media type and attributes from a Telegram Document
func (h *NewsUpdateHandler) extractDocumentMetadata(doc *tg.Document) domain.MediaMetadata {
	meta := domain.MediaMetadata{
		Type: domain.MediaTypeDocument, // Default
	}

	// First pass: check for video attributes (higher priority)
	for _, attr := range doc.Attributes {
		if a, ok := attr.(*tg.DocumentAttributeVideo); ok {
			meta.Width = a.W
			meta.Height = a.H
			meta.Duration = int(a.Duration)
			if a.RoundMessage {
				meta.Type = domain.MediaTypeVideoNote
			} else {
				meta.Type = domain.MediaTypeVideo
			}
			return meta // Video attributes have priority, return immediately
		}
	}

	// Second pass: check for audio attributes (only if not video)
	for _, attr := range doc.Attributes {
		if a, ok := attr.(*tg.DocumentAttributeAudio); ok {
			meta.Duration = int(a.Duration)
			if a.Voice {
				meta.Type = domain.MediaTypeVoice
			} else {
				meta.Type = domain.MediaTypeAudio
			}
			return meta
		}
	}

	return meta
}

// processMediaToS3 downloads media from Telegram and uploads to S3
// Returns S3 URL on success, empty string on failure
func (h *NewsUpdateHandler) processMediaToS3(ctx context.Context, media tg.MessageMediaClass, channelID string, messageID int, mediaType string) string {
	downloadFn := h.getMediaDownloadFunc()
	if downloadFn == nil {
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Str("media_type", mediaType).
			Msg("media download function not set, skipping media")
		return ""
	}

	if h.mediaUploader == nil {
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Str("media_type", mediaType).
			Msg("media uploader not set, skipping media")
		return ""
	}

	// Download media from Telegram (with FILE_REFERENCE_EXPIRED retry support)
	data, filename, contentType, err := downloadFn(ctx, media, channelID, messageID)
	if err != nil {
		h.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Str("media_type", mediaType).
			Msg("failed to download media from Telegram, skipping")
		return ""
	}

	if len(data) == 0 {
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Str("media_type", mediaType).
			Msg("downloaded media is empty, skipping")
		return ""
	}

	// Upload to S3
	url, err := h.mediaUploader.UploadMedia(ctx, channelID, messageID, filename, contentType, data)
	if err != nil {
		h.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Int("message_id", messageID).
			Str("media_type", mediaType).
			Msg("failed to upload media to S3, skipping")
		return ""
	}

	h.logger.Debug().
		Str("channel_id", channelID).
		Int("message_id", messageID).
		Str("media_type", mediaType).
		Str("url", url).
		Int("size", len(data)).
		Msg("uploaded media to S3")

	return url
}

// OnDeleteChannelMessages handles channel message deletion updates
func (h *NewsUpdateHandler) OnDeleteChannelMessages(
	ctx context.Context,
	e tg.Entities,
	u *tg.UpdateDeleteChannelMessages,
) error {
	h.lastUpdateTime.Store(time.Now())

	channelID := fmt.Sprintf("%d", u.ChannelID)

	// Try to get channel username if available in entities cache
	if channel, ok := e.Channels[u.ChannelID]; ok {
		if channel.Username != "" {
			channelID = "@" + channel.Username
		}
	}

	// Check if we're subscribed to this channel
	exists, err := h.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to check if channel exists")
		return nil
	}

	// Fallback: if channel not found by username, try to find by numeric ID
	if !exists {
		savedChannel, lookupErr := h.channelRepo.GetChannelByNumericID(ctx, u.ChannelID)
		if lookupErr == nil && savedChannel != nil {
			channelID = savedChannel.ChannelID
			exists = true
			h.logger.Debug().
				Int64("numeric_id", u.ChannelID).
				Str("channel_id", channelID).
				Msg("found channel by numeric ID fallback")
		}
	}

	if !exists {
		h.logger.Debug().
			Str("channel_id", channelID).
			Ints("message_ids", u.Messages).
			Msg("received delete event from unsubscribed channel, ignoring")
		return nil
	}

	if len(u.Messages) == 0 {
		return nil
	}

	if err := h.producer.SendNewsDeleted(ctx, channelID, u.Messages); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", channelID).
			Ints("message_ids", u.Messages).
			Msg("failed to send news deleted event to Kafka")
		h.metrics.RecordKafkaError("send_deleted_failed")
		return nil
	}

	h.logger.Info().
		Str("channel_id", channelID).
		Ints("message_ids", u.Messages).
		Msg("news deleted event sent to Kafka")

	return nil
}

// OnEditChannelMessage handles channel message edit updates
func (h *NewsUpdateHandler) OnEditChannelMessage(
	ctx context.Context,
	e tg.Entities,
	u *tg.UpdateEditChannelMessage,
) error {
	h.lastUpdateTime.Store(time.Now())

	msg, ok := u.Message.(*tg.Message)
	if !ok {
		h.logger.Debug().Msg("skipping non-message edit update")
		return nil
	}

	channelID, channelName := h.extractChannelInfo(msg, e)
	if channelID == "" {
		h.logger.Debug().Int("message_id", msg.ID).Msg("could not extract channel ID from edited message")
		return nil
	}

	// Check if we're subscribed to this channel
	exists, err := h.channelRepo.ChannelExists(ctx, channelID)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to check if channel exists")
		return nil
	}

	if !exists {
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", msg.ID).
			Msg("received edit event from unsubscribed channel, ignoring")
		return nil
	}

	// Get channel name from repository if not available
	if channelName == "" {
		channel, err := h.channelRepo.GetChannel(ctx, channelID)
		if err == nil && channel != nil {
			channelName = channel.ChannelName
		}
	}

	newsItem := h.convertToNewsItem(ctx, msg, channelID, channelName)

	if err := h.producer.SendNewsEdited(ctx, &newsItem); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", msg.ID).
			Msg("failed to send news edited event to Kafka")
		h.metrics.RecordKafkaError("send_edited_failed")
		return nil
	}

	h.logger.Info().
		Str("channel_id", channelID).
		Str("channel_name", channelName).
		Int("message_id", msg.ID).
		Int("content_length", len(newsItem.Content)).
		Msg("news edited event sent to Kafka")

	return nil
}

// IsHealthy returns true if the handler is receiving updates
func (h *NewsUpdateHandler) IsHealthy() bool {
	return h.healthy.Load()
}

// SetHealthy sets the health status of the handler
func (h *NewsUpdateHandler) SetHealthy(healthy bool) {
	h.healthy.Store(healthy)
}

// GetLastUpdateTime returns the time of the last received update
func (h *NewsUpdateHandler) GetLastUpdateTime() time.Time {
	if t, ok := h.lastUpdateTime.Load().(time.Time); ok {
		return t
	}
	return time.Time{}
}

// GetProcessedCount returns the total number of processed messages
func (h *NewsUpdateHandler) GetProcessedCount() int64 {
	return h.processedCount.Load()
}
