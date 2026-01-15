package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
)

// MTProtoClient implements domain.TelegramClient using gotd/td library
type MTProtoClient struct {
	// Telegram client instance
	client *telegram.Client

	// API credentials
	apiID   int
	apiHash string

	// Session storage (one of these will be used)
	sessionStorage   *FileSessionStorage      // Legacy file-based storage
	postgresStorage  *PostgresSessionStorage  // PostgreSQL-based storage
	phoneNumber      string

	// Connection state
	connected     bool
	disconnecting bool
	mu            sync.RWMutex
	cancelFunc    context.CancelFunc
	runDone       chan struct{} // Signals when client.Run() completes

	// Logger
	logger zerolog.Logger

	// API client for making requests
	api *tg.Client

	// Rate limiter for API calls
	rateLimiter *rate.Limiter

	// Channel info cache with expiration
	channelInfoCache      map[string]*cachedChannelInfo
	channelInfoCacheMu    sync.RWMutex
	channelInfoCacheTTL   time.Duration
}

// cachedChannelInfo represents a cached channel info entry with expiration
type cachedChannelInfo struct {
	info      *domain.ChannelInfo
	expiresAt time.Time
}

// MTProtoClientConfig holds configuration for MTProtoClient
type MTProtoClientConfig struct {
	APIID       int
	APIHash     string
	PhoneNumber string
	SessionDir  string
	Logger      zerolog.Logger
}

// NewMTProtoClient creates a new MTProto client instance
func NewMTProtoClient(cfg MTProtoClientConfig) (*MTProtoClient, error) {
	if cfg.APIID == 0 {
		return nil, fmt.Errorf("APIID is required")
	}
	if cfg.APIHash == "" {
		return nil, fmt.Errorf("APIHash is required")
	}
	if cfg.PhoneNumber == "" {
		return nil, fmt.Errorf("PhoneNumber is required")
	}
	if cfg.SessionDir == "" {
		cfg.SessionDir = "./sessions"
	}

	// Create session storage with hashed phone number
	sessionStorage, err := NewFileSessionStorage(cfg.SessionDir, cfg.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create session storage: %w", err)
	}

	client := &MTProtoClient{
		apiID:               cfg.APIID,
		apiHash:             cfg.APIHash,
		phoneNumber:         cfg.PhoneNumber,
		sessionStorage:      sessionStorage,
		logger:              cfg.Logger.With().Str("component", "mtproto_client").Str("phone", cfg.PhoneNumber).Logger(),
		connected:           false,
		rateLimiter:         rate.NewLimiter(rate.Every(time.Second), 10), // 10 requests per second
		channelInfoCache:    make(map[string]*cachedChannelInfo),
		channelInfoCacheTTL: 15 * time.Minute, // Cache channel info for 15 minutes
	}

	return client, nil
}

// NewMTProtoClientWithDB creates a new MTProto client instance with PostgreSQL session storage
func NewMTProtoClientWithDB(cfg MTProtoClientConfig, db *gorm.DB) (*MTProtoClient, error) {
	if cfg.APIID == 0 {
		return nil, fmt.Errorf("APIID is required")
	}
	if cfg.APIHash == "" {
		return nil, fmt.Errorf("APIHash is required")
	}
	if cfg.PhoneNumber == "" {
		return nil, fmt.Errorf("PhoneNumber is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Create PostgreSQL session storage
	postgresStorage, err := NewPostgresSessionStorage(db, cfg.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres session storage: %w", err)
	}

	client := &MTProtoClient{
		apiID:               cfg.APIID,
		apiHash:             cfg.APIHash,
		phoneNumber:         cfg.PhoneNumber,
		postgresStorage:     postgresStorage,
		logger:              cfg.Logger.With().Str("component", "mtproto_client").Str("phone", cfg.PhoneNumber).Logger(),
		connected:           false,
		rateLimiter:         rate.NewLimiter(rate.Every(time.Second), 10),
		channelInfoCache:    make(map[string]*cachedChannelInfo),
		channelInfoCacheTTL: 15 * time.Minute,
	}

	return client, nil
}

// getSessionStorage returns the appropriate session storage (postgres or file)
func (c *MTProtoClient) getSessionStorage() session.Storage {
	if c.postgresStorage != nil {
		return c.postgresStorage
	}
	return c.sessionStorage
}

// Connect connects to Telegram using MTProto with full authentication support
// The caller should provide a context with timeout to prevent indefinite hanging.
// Recommended timeout: 2-5 minutes to allow for user authentication input.
// If authentication is required, the user will be prompted for:
// - Verification code (sent via Telegram)
// - 2FA password (if enabled)
func (c *MTProtoClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		c.logger.Debug().Msg("already connected")
		return nil
	}
	if c.disconnecting {
		c.mu.Unlock()
		return fmt.Errorf("disconnect in progress, cannot connect")
	}
	// Keep the lock to prevent concurrent connection attempts
	defer c.mu.Unlock()

	c.logger.Info().Msg("connecting to Telegram")

	// Create telegram client with session storage
	c.client = telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: c.getSessionStorage(),
	})

	// Create cancellable context for client lifecycle
	clientCtx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	// Channel to signal when connection is ready
	readyChan := make(chan struct{})
	errChan := make(chan error, 1)
	started := make(chan struct{})
	c.runDone = make(chan struct{})

	// Start the client in a goroutine
	go func() {
		defer close(c.runDone) // Signal when Run() completes
		close(started)
		err := c.client.Run(clientCtx, func(ctx context.Context) error {
			// Get API client
			c.api = c.client.API()

			// Check authorization status
			status, err := c.client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to check auth status: %w", err)
			}

			// If not authorized, perform authentication with retry logic
			if !status.Authorized {
				c.logger.Info().Msg("not authorized, starting authentication")
				if err := c.authenticateWithRetry(ctx, 3); err != nil {
					c.logger.Error().Err(err).Msg("authentication failed")
					return domain.ErrAuthenticationFailed
				}
			} else {
				c.logger.Info().Msg("session restored from storage")
			}

			// Set connected state
			c.connected = true
			c.logger.Info().Msg("successfully connected to Telegram")

			// Signal that connection is ready
			close(readyChan)

			// Keep connection alive
			<-ctx.Done()
			return ctx.Err()
		})
		// Always send error to channel, even if nil
		select {
		case errChan <- err:
		default:
		}
	}()

	// Ensure goroutine has started
	<-started

	// Wait for connection to be fully ready or error
	select {
	case <-readyChan:
		return nil
	case err := <-errChan:
		// Cancel to clean up goroutine
		cancel()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		return nil
	case <-ctx.Done():
		// Cancel to clean up goroutine
		cancel()
		return ctx.Err()
	}
}

// Disconnect disconnects from Telegram with graceful shutdown
// The operation respects a 10-second timeout for cleanup operations.
// The session is automatically saved by the underlying gotd/td client before shutdown.
// Multiple calls to Disconnect() are safe and will return nil if already disconnected.
// This method is safe for concurrent use.
func (c *MTProtoClient) Disconnect(ctx context.Context) error {
	c.mu.Lock()

	// Check if already disconnecting
	if c.disconnecting {
		c.mu.Unlock()
		c.logger.Debug().Msg("disconnect already in progress")
		return nil
	}

	// Check if already disconnected
	if !c.connected {
		c.mu.Unlock()
		c.logger.Debug().Msg("already disconnected")
		return nil
	}

	c.logger.Info().Msg("disconnecting from Telegram")

	// Mark as disconnecting
	c.disconnecting = true
	cancelFunc := c.cancelFunc
	runDone := c.runDone
	c.mu.Unlock()

	// Cancel the client context to stop the goroutine
	if cancelFunc != nil {
		c.logger.Debug().Msg("cancelling client context")
		cancelFunc()

		// Wait for client.Run() goroutine to actually finish
		if runDone != nil {
			select {
			case <-runDone:
				c.logger.Debug().Msg("client stopped gracefully")
			case <-ctx.Done():
				c.logger.Warn().Msg("disconnect timeout reached while waiting for client shutdown")
				// Don't return error yet, still clean up state
			}
		}
	}

	// Clean up state
	c.mu.Lock()
	c.client = nil
	c.api = nil
	c.connected = false
	c.cancelFunc = nil
	c.runDone = nil
	c.disconnecting = false
	c.mu.Unlock()

	c.logger.Info().Msg("successfully disconnected from Telegram")
	return nil
}

// IsConnected checks if client is connected to Telegram
func (c *MTProtoClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetAccountID returns the phone number as unique account identifier
func (c *MTProtoClient) GetAccountID() string {
	return c.phoneNumber
}

// validateChannelID validates channel ID format
func validateChannelID(channelID string) error {
	if channelID == "" {
		return domain.ErrInvalidChannelID
	}
	// Channel ID should be username (@channel) or numeric ID
	if !strings.HasPrefix(channelID, "@") && !isNumeric(channelID) {
		return domain.ErrInvalidChannelID
	}
	return nil
}

// isNumeric checks if string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// resolveChannel resolves a channel ID to InputChannel
// Only supports @username format, not numeric IDs (which require access hash)
func (c *MTProtoClient) resolveChannel(ctx context.Context, channelID string) (*tg.InputChannel, error) {
	if !strings.HasPrefix(channelID, "@") {
		return nil, fmt.Errorf("resolving by numeric ID requires access hash, use @username format")
	}

	username := strings.TrimPrefix(channelID, "@")
	resolved, err := c.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to resolve channel")
		return nil, fmt.Errorf("failed to resolve channel: %w", err)
	}

	// Extract channel from resolved peer
	for _, chat := range resolved.Chats {
		if channel, ok := chat.(*tg.Channel); ok {
			return &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}, nil
		}
	}

	return nil, fmt.Errorf("resolved peer is not a channel")
}

// resolveChannelWithInfo resolves a channel and extracts both InputChannel and channel name
// This combines channel resolution and info retrieval to avoid duplicate API calls
func (c *MTProtoClient) resolveChannelWithInfo(ctx context.Context, api *tg.Client, channelID string) (*tg.InputChannel, string, error) {
	if !strings.HasPrefix(channelID, "@") {
		return nil, "", fmt.Errorf("resolving by numeric ID requires access hash, use @username format")
	}

	username := strings.TrimPrefix(channelID, "@")
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to resolve channel")
		return nil, "", fmt.Errorf("failed to resolve channel: %w", err)
	}

	// Extract channel from resolved peer
	for _, chat := range resolved.Chats {
		if channel, ok := chat.(*tg.Channel); ok {
			inputChannel := &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}
			channelName := channel.Title
			if channelName == "" {
				channelName = channelID
			}
			return inputChannel, channelName, nil
		}
	}

	return nil, "", fmt.Errorf("resolved peer is not a channel")
}

// JoinChannel joins a Telegram channel with retry mechanism for FloodWait
// Supports both @username and numeric channel ID formats
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 60 seconds to allow for potential flood wait delays.
// Returns specific errors: ErrChannelNotFound, ErrChannelPrivate, ErrPeerFlood, ErrFloodWait
func (c *MTProtoClient) JoinChannel(ctx context.Context, channelID string) error {
	// Validate channel ID
	if err := validateChannelID(channelID); err != nil {
		return err
	}

	c.mu.RLock()
	if !c.connected || c.api == nil {
		c.mu.RUnlock()
		return domain.ErrNotConnected
	}
	c.mu.RUnlock()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	c.logger.Info().Str("channel_id", channelID).Msg("joining channel")

	// Resolve the channel username to get InputChannel
	inputChannel, err := c.resolveChannel(ctx, channelID)
	if err != nil {
		return err
	}

	// Join with retry mechanism for FloodWait
	return c.joinChannelWithRetry(ctx, channelID, inputChannel, 3)
}

// joinChannelWithRetry attempts to join a channel with retry logic for FloodWait errors
func (c *MTProtoClient) joinChannelWithRetry(ctx context.Context, channelID string, inputChannel *tg.InputChannel, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Join the channel
		_, err := c.api.ChannelsJoinChannel(ctx, inputChannel)
		if err == nil {
			c.logger.Info().Str("channel_id", channelID).Msg("successfully joined channel")
			return nil
		}

		lastErr = err

		// Handle specific Telegram errors
		if tgerr.Is(err, "CHANNEL_PRIVATE") {
			c.logger.Error().Str("channel_id", channelID).Msg("channel is private")
			return domain.ErrChannelPrivate
		}

		if tgerr.Is(err, "CHANNEL_INVALID") || tgerr.Is(err, "USERNAME_INVALID") || tgerr.Is(err, "USERNAME_NOT_OCCUPIED") {
			c.logger.Error().Str("channel_id", channelID).Msg("channel not found")
			return domain.ErrChannelNotFound
		}

		if tgerr.Is(err, "CHANNELS_TOO_MUCH") {
			c.logger.Error().Msg("joined too many channels")
			return fmt.Errorf("joined too many channels, cannot join more")
		}

		if tgerr.Is(err, "USER_BANNED_IN_CHANNEL") {
			c.logger.Error().Str("channel_id", channelID).Msg("user is banned in this channel")
			return fmt.Errorf("user is banned in channel")
		}

		// Handle PEER_FLOOD - anti-spam restriction (non-retryable)
		if tgerr.Is(err, "PEER_FLOOD") {
			c.logger.Error().Str("channel_id", channelID).Msg("peer flood detected - too many join requests")
			return domain.ErrPeerFlood
		}

		// Handle FloodWait - temporary rate limit (retryable)
		var floodErr *tgerr.Error
		if errors.As(err, &floodErr) && floodErr.Code == 420 {
			waitDuration := time.Duration(floodErr.Argument) * time.Second
			c.logger.Warn().
				Str("channel_id", channelID).
				Int("attempt", attempt+1).
				Dur("wait_duration", waitDuration).
				Msg("flood wait detected, waiting before retry")

			// Check if we have time to wait
			select {
			case <-time.After(waitDuration):
				// Continue to next retry
				continue
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during flood wait: %w", ctx.Err())
			}
		}

		// For other errors, log and retry
		c.logger.Warn().
			Err(err).
			Str("channel_id", channelID).
			Int("attempt", attempt+1).
			Msg("failed to join channel, retrying")

		// Short delay before retry
		select {
		case <-time.After(time.Second):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	c.logger.Error().Err(lastErr).Str("channel_id", channelID).Msg("failed to join channel after retries")
	return fmt.Errorf("failed to join channel after %d attempts: %w", maxRetries, lastErr)
}

// LeaveChannel leaves a Telegram channel
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30 seconds for normal operations, longer for slow networks.
// Returns specific errors: ErrChannelNotFound, ErrChannelPrivate
func (c *MTProtoClient) LeaveChannel(ctx context.Context, channelID string) error {
	// Validate channel ID
	if err := validateChannelID(channelID); err != nil {
		return err
	}

	c.mu.RLock()
	if !c.connected || c.api == nil {
		c.mu.RUnlock()
		return domain.ErrNotConnected
	}
	c.mu.RUnlock()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	c.logger.Info().Str("channel_id", channelID).Msg("leaving channel")

	// Resolve the channel username
	inputChannel, err := c.resolveChannel(ctx, channelID)
	if err != nil {
		// Check if resolution failed due to channel not found
		if tgerr.Is(err, "USERNAME_INVALID") || tgerr.Is(err, "USERNAME_NOT_OCCUPIED") {
			c.logger.Error().Str("channel_id", channelID).Msg("channel not found during resolution")
			return domain.ErrChannelNotFound
		}
		return err
	}

	// Leave the channel
	_, err = c.api.ChannelsLeaveChannel(ctx, inputChannel)
	if err != nil {
		// Handle specific Telegram errors
		if tgerr.Is(err, "CHANNEL_INVALID") || tgerr.Is(err, "CHANNEL_NOT_FOUND") {
			c.logger.Error().Str("channel_id", channelID).Msg("channel not found")
			return domain.ErrChannelNotFound
		}

		if tgerr.Is(err, "CHANNEL_PRIVATE") {
			c.logger.Error().Str("channel_id", channelID).Msg("channel is private")
			return domain.ErrChannelPrivate
		}

		if tgerr.Is(err, "USER_NOT_PARTICIPANT") {
			c.logger.Warn().Str("channel_id", channelID).Msg("user is not a participant of this channel")
			// Return success since the end result is the same - user is not in the channel
			return nil
		}

		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to leave channel")
		return fmt.Errorf("failed to leave channel: %w", err)
	}

	c.logger.Info().Str("channel_id", channelID).Msg("successfully left channel")
	return nil
}

// GetChannelMessages retrieves recent messages from a channel with pagination support
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30-60 seconds depending on message count and network conditions.
// offset parameter skips the first N messages (0 means no offset)
func (c *MTProtoClient) GetChannelMessages(ctx context.Context, channelID string, limit, offset int) ([]domain.NewsItem, error) {
	// Validate channel ID
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}

	// Validate limit and offset with upper bounds
	if limit <= 0 {
		limit = 10 // Default limit
	} else if limit > 100 {
		limit = 100 // Maximum limit to prevent abuse and excessive API usage
	}
	if offset < 0 {
		offset = 0
	}

	c.mu.RLock()
	if !c.connected || c.api == nil {
		c.mu.RUnlock()
		return nil, domain.ErrNotConnected
	}
	// Capture api reference while holding lock to prevent race conditions
	api := c.api
	c.mu.RUnlock()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	c.logger.Debug().
		Str("channel_id", channelID).
		Int("limit", limit).
		Int("offset", offset).
		Msg("fetching channel messages")

	// Resolve the channel and get channel info in one operation
	inputChannel, channelName, err := c.resolveChannelWithInfo(ctx, api, channelID)
	if err != nil {
		return nil, err
	}

	// Calculate safe fetch limit with upper bound to prevent excessive API usage
	// For small offsets, fetch offset + limit messages and slice locally
	// For large offsets (>= 100), return empty to avoid API abuse
	maxFetch := limit + offset
	if offset >= 100 {
		c.logger.Debug().
			Str("channel_id", channelID).
			Int("offset", offset).
			Msg("offset too large, returning empty results")
		return []domain.NewsItem{}, nil
	}
	if maxFetch > 100 {
		maxFetch = 100
	}

	// Get messages from the channel with offset support
	messagesRequest := &tg.MessagesGetHistoryRequest{
		Peer:     &tg.InputPeerChannel{ChannelID: inputChannel.GetChannelID(), AccessHash: inputChannel.GetAccessHash()},
		OffsetID: 0,
		Limit:    maxFetch,
	}

	result, err := api.MessagesGetHistory(ctx, messagesRequest)
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to get messages")
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Parse messages using helper function
	var messageSlice []tg.MessageClass
	switch messages := result.(type) {
	case *tg.MessagesChannelMessages:
		messageSlice = messages.Messages
	case *tg.MessagesMessages:
		messageSlice = messages.Messages
	default:
		c.logger.Warn().Str("channel_id", channelID).Str("type", fmt.Sprintf("%T", result)).Msg("unexpected message type")
		return []domain.NewsItem{}, nil
	}

	newsItems := c.processMessagesSlice(messageSlice, channelID, channelName, offset, limit)

	c.logger.Debug().
		Str("channel_id", channelID).
		Int("messages_count", len(newsItems)).
		Msg("fetched messages")
	return newsItems, nil
}

// processMessagesSlice processes a slice of Telegram messages and converts them to NewsItems
// This helper function eliminates code duplication between different message type handlers
func (c *MTProtoClient) processMessagesSlice(messages []tg.MessageClass, channelID, channelName string, offset, limit int) []domain.NewsItem {
	// Handle empty message list
	if len(messages) == 0 {
		c.logger.Debug().Str("channel_id", channelID).Msg("no messages to process")
		return []domain.NewsItem{}
	}

	// Apply offset by skipping first N messages
	startIdx := offset
	if startIdx >= len(messages) {
		c.logger.Debug().
			Str("channel_id", channelID).
			Int("offset", offset).
			Int("total_messages", len(messages)).
			Msg("offset exceeds message count")
		return []domain.NewsItem{}
	}

	messagesToProcess := messages[startIdx:]
	if len(messagesToProcess) > limit {
		messagesToProcess = messagesToProcess[:limit]
	}

	// Pre-allocate slice with expected capacity for better performance
	newsItems := make([]domain.NewsItem, 0, len(messagesToProcess))

	for _, msg := range messagesToProcess {
		// Handle regular messages
		if message, ok := msg.(*tg.Message); ok {
			newsItem := domain.NewsItem{
				ChannelID:   channelID,
				ChannelName: channelName,
				MessageID:   message.ID,
				Content:     message.Message,
				MediaURLs:   []string{},
				Date:        time.Unix(int64(message.Date), 0),
			}

			// Extract media URLs if present
			if message.Media != nil {
				newsItem.MediaURLs = c.extractMediaURLs(message.Media)
			}

			newsItems = append(newsItems, newsItem)
		} else if _, ok := msg.(*tg.MessageEmpty); ok {
			// Handle deleted/empty messages - skip them
			c.logger.Debug().
				Str("channel_id", channelID).
				Msg("skipping deleted/empty message")
			continue
		} else if msgService, ok := msg.(*tg.MessageService); ok {
			// Handle service messages - skip them (channel created, user joined, etc.)
			c.logger.Debug().
				Str("channel_id", channelID).
				Int("message_id", msgService.ID).
				Msg("skipping service message")
			continue
		}
	}

	return newsItems
}

// extractMediaURLs extracts media URLs from different message media types
//
// For Telegram-hosted media (photos, videos, documents), returns pseudo-URLs
// in the format "type://id" or "type://id:filename" since actual downloads
// require additional file references and access hashes not stored here.
//
// Supported URL schemes:
//   - photo://[id] - Photo media (use ID with Telegram API to download)
//   - video://[id][:filename] - Video files
//   - audio://[id][:filename] - Audio files
//   - document://[id][:filename] - Generic documents
//   - http(s)://... - Web URLs from MessageMediaWebPage
//   - geo:[lat],[long] - Geographic coordinates (RFC 5870 format)
//   - contact://tel:[phone] - Contact information
//
// To download Telegram media, consumers must use the Telegram API with
// the extracted ID plus the file's access hash and file reference.
//
// Returns empty slice if no media is present or media type is unsupported (e.g., polls).
func (c *MTProtoClient) extractMediaURLs(media tg.MessageMediaClass) []string {
	var urls []string

	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		// Extract photo URL
		if photo, ok := m.Photo.(*tg.Photo); ok {
			// Build a reference URL using photo ID
			// Note: Actual download would require file reference and access hash
			photoURL := fmt.Sprintf("photo://%d", photo.ID)
			urls = append(urls, photoURL)
		}

	case *tg.MessageMediaDocument:
		// Extract document/video/audio URL
		if doc, ok := m.Document.(*tg.Document); ok {
			// Determine document type from attributes
			docType := "document"
			var fileName string

			for _, attr := range doc.Attributes {
				switch a := attr.(type) {
				case *tg.DocumentAttributeVideo:
					docType = "video"
				case *tg.DocumentAttributeAudio:
					docType = "audio"
				case *tg.DocumentAttributeFilename:
					fileName = a.FileName
				}
			}

			// Build a reference URL
			docURL := fmt.Sprintf("%s://%d", docType, doc.ID)
			if fileName != "" {
				docURL = fmt.Sprintf("%s:%s", docURL, fileName)
			}
			urls = append(urls, docURL)
		}

	case *tg.MessageMediaWebPage:
		// Extract webpage URL if present
		if webpage, ok := m.Webpage.(*tg.WebPage); ok {
			if webpage.URL != "" {
				urls = append(urls, webpage.URL)
			}
			// Also extract photo from webpage if present
			if webpage.Photo != nil {
				if photo, ok := webpage.Photo.(*tg.Photo); ok {
					photoURL := fmt.Sprintf("photo://%d", photo.ID)
					urls = append(urls, photoURL)
				}
			}
			// Extract document from webpage if present
			if webpage.Document != nil {
				if doc, ok := webpage.Document.(*tg.Document); ok {
					docURL := fmt.Sprintf("document://%d", doc.ID)
					urls = append(urls, docURL)
				}
			}
		}

	case *tg.MessageMediaGeo:
		// Extract geo location as URL
		if m.Geo != nil {
			if geoPoint, ok := m.Geo.(*tg.GeoPoint); ok {
				geoURL := fmt.Sprintf("geo:%.6f,%.6f", geoPoint.Lat, geoPoint.Long)
				urls = append(urls, geoURL)
			}
		}

	case *tg.MessageMediaContact:
		// Extract contact info as vCard-style URL
		contactURL := fmt.Sprintf("contact://tel:%s", m.PhoneNumber)
		urls = append(urls, contactURL)

	case *tg.MessageMediaPoll:
		// Polls don't have URLs, skip
		c.logger.Debug().Msg("skipping poll media (no URL)")

	default:
		// Unknown media type
		c.logger.Debug().Str("media_type", fmt.Sprintf("%T", m)).Msg("unknown media type")
	}

	return urls
}

// GetChannelInfo retrieves detailed information about a channel
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30 seconds for normal operations, longer for slow networks.
// Returns detailed channel information including title, description, participant count, etc.
// Uses caching with 15-minute TTL to reduce API calls.
func (c *MTProtoClient) GetChannelInfo(ctx context.Context, channelID string) (*domain.ChannelInfo, error) {
	// Validate channel ID
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}

	// Check cache first
	c.channelInfoCacheMu.RLock()
	if cached, ok := c.channelInfoCache[channelID]; ok {
		if time.Now().Before(cached.expiresAt) {
			c.channelInfoCacheMu.RUnlock()
			c.logger.Debug().Str("channel_id", channelID).Msg("returning cached channel info")
			return cached.info, nil
		}
		// Cache expired, will fetch fresh data
	}
	c.channelInfoCacheMu.RUnlock()

	c.mu.RLock()
	if !c.connected || c.api == nil {
		c.mu.RUnlock()
		return nil, domain.ErrNotConnected
	}
	api := c.api
	c.mu.RUnlock()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	c.logger.Debug().Str("channel_id", channelID).Msg("fetching channel info from API")

	// Resolve the channel
	username := strings.TrimPrefix(channelID, "@")
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		// Handle Telegram-specific errors
		if tgerr.Is(err, "USERNAME_INVALID") || tgerr.Is(err, "USERNAME_NOT_OCCUPIED") {
			c.logger.Error().Str("channel_id", channelID).Msg("channel not found")
			return nil, domain.ErrChannelNotFound
		}
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to resolve channel")
		return nil, fmt.Errorf("failed to resolve channel: %w", err)
	}

	// Find channel in resolved chats
	var channel *tg.Channel
	for _, chat := range resolved.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			channel = ch
			break
		}
	}

	if channel == nil {
		return nil, fmt.Errorf("resolved peer is not a channel")
	}

	// Check if channel is accessible
	if channel.Left {
		c.logger.Warn().Str("channel_id", channelID).Msg("channel was left")
	}

	// Build ChannelInfo
	info := &domain.ChannelInfo{
		ID:           fmt.Sprintf("%d", channel.ID),
		Username:     channel.Username,
		Title:        channel.Title,
		IsVerified:   channel.Verified,
		IsRestricted: channel.Restricted,
		CreatedAt:    time.Unix(int64(channel.Date), 0),
	}

	// Get full channel information including description and participant count
	fullChannel, err := c.getFullChannelInfo(ctx, api, channel)
	if err != nil {
		// Log error but cache and return basic info
		c.logger.Warn().Err(err).Str("channel_id", channelID).Msg("failed to get full channel info, returning basic info")

		// Cache basic info
		c.cacheChannelInfo(channelID, info)
		return info, nil
	}

	// Update with full channel information
	if fullChannel.About != "" {
		info.About = fullChannel.About
	}
	if fullChannel.ParticipantsCount != 0 {
		info.ParticipantsCount = fullChannel.ParticipantsCount
	}

	c.logger.Debug().
		Str("channel_id", channelID).
		Str("title", info.Title).
		Int("participants", info.ParticipantsCount).
		Msg("successfully fetched channel info")

	// Cache full channel info
	c.cacheChannelInfo(channelID, info)

	return info, nil
}

// cacheChannelInfo stores channel info in cache with TTL
func (c *MTProtoClient) cacheChannelInfo(channelID string, info *domain.ChannelInfo) {
	c.channelInfoCacheMu.Lock()
	defer c.channelInfoCacheMu.Unlock()

	c.channelInfoCache[channelID] = &cachedChannelInfo{
		info:      info,
		expiresAt: time.Now().Add(c.channelInfoCacheTTL),
	}
}

// getFullChannelInfo retrieves full channel information using channels.getFullChannel
func (c *MTProtoClient) getFullChannelInfo(ctx context.Context, api *tg.Client, channel *tg.Channel) (*tg.ChannelFull, error) {
	inputChannel := &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	}

	fullChan, err := api.ChannelsGetFullChannel(ctx, inputChannel)
	if err != nil {
		// Handle specific errors
		if tgerr.Is(err, "CHANNEL_PRIVATE") {
			return nil, domain.ErrChannelPrivate
		}
		if tgerr.Is(err, "CHANNEL_INVALID") {
			return nil, domain.ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to get full channel: %w", err)
	}

	// Extract ChannelFull from response
	if channelFull, ok := fullChan.FullChat.(*tg.ChannelFull); ok {
		return channelFull, nil
	}

	return nil, fmt.Errorf("unexpected full chat type: %T", fullChan.FullChat)
}

// Ensure MTProtoClient implements domain.TelegramClient interface
var _ domain.TelegramClient = (*MTProtoClient)(nil)
