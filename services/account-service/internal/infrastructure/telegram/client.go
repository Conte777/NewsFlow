package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
)

// MTProtoClient implements domain.TelegramClient using gotd/td library
type MTProtoClient struct {
	// Telegram client instance
	client *telegram.Client

	// API credentials
	apiID   int
	apiHash string

	// Session storage
	sessionStorage *FileSessionStorage
	phoneNumber    string

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
}

// MTProtoClientConfig holds configuration for MTProtoClient
type MTProtoClientConfig struct {
	APIID       int
	APIHash     string
	PhoneNumber string
	SessionDir  string
	Logger      zerolog.Logger
}

// maskPhoneNumber masks phone number for logging (keeps first 2 and last 2 digits)
func maskPhoneNumber(phone string) string {
	if len(phone) < 4 {
		return "***"
	}
	return phone[:2] + strings.Repeat("*", len(phone)-4) + phone[len(phone)-2:]
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

	maskedPhone := maskPhoneNumber(cfg.PhoneNumber)

	client := &MTProtoClient{
		apiID:          cfg.APIID,
		apiHash:        cfg.APIHash,
		phoneNumber:    cfg.PhoneNumber,
		sessionStorage: sessionStorage,
		logger:         cfg.Logger.With().Str("component", "mtproto_client").Str("phone", maskedPhone).Logger(),
		connected:      false,
		rateLimiter:    rate.NewLimiter(rate.Every(time.Second), 10), // 10 requests per second
	}

	return client, nil
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
		SessionStorage: c.sessionStorage,
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

// JoinChannel joins a Telegram channel
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30 seconds for normal operations, longer for slow networks.
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

	// Join the channel
	_, err = c.api.ChannelsJoinChannel(ctx, inputChannel)
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to join channel")
		return fmt.Errorf("failed to join channel: %w", err)
	}

	c.logger.Info().Str("channel_id", channelID).Msg("successfully joined channel")
	return nil
}

// LeaveChannel leaves a Telegram channel
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30 seconds for normal operations, longer for slow networks.
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
		return err
	}

	// Leave the channel
	_, err = c.api.ChannelsLeaveChannel(ctx, inputChannel)
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to leave channel")
		return fmt.Errorf("failed to leave channel: %w", err)
	}

	c.logger.Info().Str("channel_id", channelID).Msg("successfully left channel")
	return nil
}

// GetChannelMessages retrieves recent messages from a channel
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30-60 seconds depending on message count and network conditions.
func (c *MTProtoClient) GetChannelMessages(ctx context.Context, channelID string, limit int) ([]domain.NewsItem, error) {
	// Validate channel ID
	if err := validateChannelID(channelID); err != nil {
		return nil, err
	}

	c.mu.RLock()
	if !c.connected || c.api == nil {
		c.mu.RUnlock()
		return nil, domain.ErrNotConnected
	}
	c.mu.RUnlock()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	c.logger.Debug().Str("channel_id", channelID).Int("limit", limit).Msg("fetching channel messages")

	// Resolve the channel
	inputChannel, err := c.resolveChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	// Get messages from the channel
	messagesRequest := &tg.MessagesGetHistoryRequest{
		Peer:  &tg.InputPeerChannel{ChannelID: inputChannel.GetChannelID(), AccessHash: inputChannel.GetAccessHash()},
		Limit: limit,
	}

	result, err := c.api.MessagesGetHistory(ctx, messagesRequest)
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to get messages")
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Parse messages
	var newsItems []domain.NewsItem
	switch messages := result.(type) {
	case *tg.MessagesChannelMessages:
		for _, msg := range messages.Messages {
			if message, ok := msg.(*tg.Message); ok {
				newsItem := domain.NewsItem{
					ChannelID: channelID,
					MessageID: message.ID,
					Content:   message.Message,
					Date:      time.Unix(int64(message.Date), 0),
				}

				// Extract media URLs if present
				if message.Media != nil {
					// Handle different media types
					// This is simplified - full implementation would handle photos, documents, etc.
					newsItem.MediaURLs = []string{} // Placeholder
				}

				newsItems = append(newsItems, newsItem)
			}
		}
	}

	c.logger.Debug().Str("channel_id", channelID).Int("messages_count", len(newsItems)).Msg("fetched messages")
	return newsItems, nil
}

// GetChannelInfo retrieves information about a channel
// The caller should provide a context with timeout to prevent hanging operations.
// Recommended timeout: 30 seconds for normal operations, longer for slow networks.
func (c *MTProtoClient) GetChannelInfo(ctx context.Context, channelID string) (string, error) {
	// Validate channel ID
	if err := validateChannelID(channelID); err != nil {
		return "", err
	}

	c.mu.RLock()
	if !c.connected || c.api == nil {
		c.mu.RUnlock()
		return "", domain.ErrNotConnected
	}
	c.mu.RUnlock()

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	c.logger.Debug().Str("channel_id", channelID).Msg("fetching channel info")

	// Resolve the channel
	username := strings.TrimPrefix(channelID, "@")
	resolved, err := c.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		c.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to resolve channel")
		return "", fmt.Errorf("failed to resolve channel: %w", err)
	}

	for _, chat := range resolved.Chats {
		if channel, ok := chat.(*tg.Channel); ok {
			return channel.Title, nil
		}
	}

	return "", fmt.Errorf("resolved peer is not a channel")
}

// Ensure MTProtoClient implements domain.TelegramClient interface
var _ domain.TelegramClient = (*MTProtoClient)(nil)
