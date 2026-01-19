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
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/telegram/fileid"
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
	producer    domain.KafkaProducer
	logger      zerolog.Logger
	metrics     *metrics.Metrics

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
	producer domain.KafkaProducer,
	logger zerolog.Logger,
	m *metrics.Metrics,
) *NewsUpdateHandler {
	h := &NewsUpdateHandler{
		channelRepo: channelRepo,
		producer:    producer,
		logger:      logger.With().Str("component", "news_update_handler").Logger(),
		metrics:     m,
		albumBuffer: make(map[int64]*pendingAlbum),
	}
	h.healthy.Store(true)
	h.lastUpdateTime.Store(time.Now())
	return h
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

	channel, err := h.channelRepo.GetChannel(ctx, channelID)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to get channel")
		return nil
	}

	if msg.ID <= channel.LastProcessedMessageID {
		h.logger.Debug().
			Str("channel_id", channelID).
			Int("message_id", msg.ID).
			Int("last_processed", channel.LastProcessedMessageID).
			Msg("skipping already processed message")
		return nil
	}

	if channelName == "" && channel.ChannelName != "" {
		channelName = channel.ChannelName
	}

	newsItem := h.convertToNewsItem(msg, channelID, channelName)

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

	// Collect all media URLs and find text content
	var allMediaURLs []string
	var textContent string

	for _, item := range items {
		allMediaURLs = append(allMediaURLs, item.MediaURLs...)
		if item.Content != "" && textContent == "" {
			textContent = item.Content // Use first non-empty content
		}
	}

	result.MediaURLs = allMediaURLs
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
	if err := h.producer.SendNewsReceived(ctx, newsItem); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", msgID).
			Msg("failed to send news to Kafka")
		h.metrics.RecordKafkaError("send_failed")
		return
	}

	if err := h.channelRepo.UpdateLastProcessedMessageID(ctx, channelID, msgID); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", channelID).
			Int("message_id", msgID).
			Msg("failed to update last processed message ID")
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
func (h *NewsUpdateHandler) convertToNewsItem(msg *tg.Message, channelID, channelName string) domain.NewsItem {
	newsItem := domain.NewsItem{
		ChannelID:   channelID,
		ChannelName: channelName,
		MessageID:   msg.ID,
		Content:     msg.Message,
		MediaURLs:   h.extractMediaURLs(msg.Media),
		Date:        time.Unix(int64(msg.Date), 0),
		GroupedID:   msg.GroupedID, // For album grouping
	}

	return newsItem
}

// extractMediaURLs extracts media from message media types
// Returns Bot API file_id strings for Telegram media, HTTP URLs for web content
func (h *NewsUpdateHandler) extractMediaURLs(media tg.MessageMediaClass) []string {
	if media == nil {
		return []string{}
	}

	var urls []string

	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		// Extract photo and encode as Bot API file_id
		if photo, ok := m.Photo.(*tg.Photo); ok {
			fileID := fileid.EncodePhotoFileID(photo)
			if fileID != "" {
				urls = append(urls, fileID)
			}
		}

	case *tg.MessageMediaDocument:
		// Extract document and encode as Bot API file_id
		if doc, ok := m.Document.(*tg.Document); ok {
			fileType := fileid.DetectDocumentType(doc)
			fileID := fileid.EncodeDocumentFileID(doc, fileType)
			if fileID != "" {
				urls = append(urls, fileID)
			}
		}

	case *tg.MessageMediaWebPage:
		if webpage, ok := m.Webpage.(*tg.WebPage); ok {
			if webpage.URL != "" {
				urls = append(urls, webpage.URL)
			}
			// Also extract photo from webpage if present
			if webpage.Photo != nil {
				if photo, ok := webpage.Photo.(*tg.Photo); ok {
					fileID := fileid.EncodePhotoFileID(photo)
					if fileID != "" {
						urls = append(urls, fileID)
					}
				}
			}
		}

	case *tg.MessageMediaGeo:
		if m.Geo != nil {
			if geoPoint, ok := m.Geo.(*tg.GeoPoint); ok {
				geoURL := fmt.Sprintf("geo:%.6f,%.6f", geoPoint.Lat, geoPoint.Long)
				urls = append(urls, geoURL)
			}
		}

	case *tg.MessageMediaContact:
		contactURL := fmt.Sprintf("contact://tel:%s", m.PhoneNumber)
		urls = append(urls, contactURL)
	}

	return urls
}

// OnDeleteChannelMessages handles channel message deletion updates
func (h *NewsUpdateHandler) OnDeleteChannelMessages(
	ctx context.Context,
	e tg.Entities,
	u *tg.UpdateDeleteChannelMessages,
) error {
	h.lastUpdateTime.Store(time.Now())

	channelID := fmt.Sprintf("%d", u.ChannelID)

	// Try to get channel username if available
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

	newsItem := h.convertToNewsItem(msg, channelID, channelName)

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
