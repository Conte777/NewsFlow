package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/rs/zerolog"

	"github.com/yourusername/telegram-news-feed/account-service/internal/domain"
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
	connected bool
	mu        sync.RWMutex

	// Logger
	logger zerolog.Logger

	// API client for making requests
	api *tg.Client
}

// MTProtoClientConfig holds configuration for MTProtoClient
type MTProtoClientConfig struct {
	APIID          int
	APIHash        string
	PhoneNumber    string
	SessionDir     string
	Logger         zerolog.Logger
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

	// Create session storage
	sessionStorage, err := NewFileSessionStorage(cfg.SessionDir, cfg.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create session storage: %w", err)
	}

	client := &MTProtoClient{
		apiID:          cfg.APIID,
		apiHash:        cfg.APIHash,
		phoneNumber:    cfg.PhoneNumber,
		sessionStorage: sessionStorage,
		logger:         cfg.Logger.With().Str("component", "mtproto_client").Str("phone", cfg.PhoneNumber).Logger(),
		connected:      false,
	}

	return client, nil
}

// Connect connects to Telegram using MTProto
func (c *MTProtoClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		c.logger.Debug().Msg("already connected")
		return nil
	}

	c.logger.Info().Msg("connecting to Telegram")

	// Create telegram client with session storage
	c.client = telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: c.sessionStorage,
	})

	// Start the client
	return c.client.Run(ctx, func(ctx context.Context) error {
		// Get API client
		c.api = c.client.API()

		// Check authorization status
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to check auth status: %w", err)
		}

		// If not authorized, we need to authenticate
		if !status.Authorized {
			c.logger.Info().Msg("not authorized, authentication required")
			// Note: Actual authentication implementation would be done separately
			// This is just the connection setup
			return fmt.Errorf("authentication required for phone %s", c.phoneNumber)
		}

		c.connected = true
		c.logger.Info().Msg("successfully connected to Telegram")
		return nil
	})
}

// Disconnect disconnects from Telegram
func (c *MTProtoClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		c.logger.Debug().Msg("already disconnected")
		return nil
	}

	c.logger.Info().Msg("disconnecting from Telegram")

	if c.client != nil {
		// The client's Run context will handle cleanup when cancelled
		c.client = nil
		c.api = nil
	}

	c.connected = false
	c.logger.Info().Msg("successfully disconnected from Telegram")
	return nil
}

// IsConnected checks if client is connected to Telegram
func (c *MTProtoClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// JoinChannel joins a Telegram channel
func (c *MTProtoClient) JoinChannel(ctx context.Context, channelID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.api == nil {
		return fmt.Errorf("client is not connected")
	}

	c.logger.Info().Str("channel_id", channelID).Msg("joining channel")

	// TODO: Implement actual channel join logic using c.api
	// This will involve resolving the channel username and joining it

	return fmt.Errorf("not implemented")
}

// LeaveChannel leaves a Telegram channel
func (c *MTProtoClient) LeaveChannel(ctx context.Context, channelID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.api == nil {
		return fmt.Errorf("client is not connected")
	}

	c.logger.Info().Str("channel_id", channelID).Msg("leaving channel")

	// TODO: Implement actual channel leave logic using c.api

	return fmt.Errorf("not implemented")
}

// GetChannelMessages retrieves recent messages from a channel
func (c *MTProtoClient) GetChannelMessages(ctx context.Context, channelID string, limit int) ([]domain.NewsItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.api == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	c.logger.Debug().Str("channel_id", channelID).Int("limit", limit).Msg("fetching channel messages")

	// TODO: Implement actual message fetching logic using c.api

	return nil, fmt.Errorf("not implemented")
}

// GetChannelInfo retrieves information about a channel
func (c *MTProtoClient) GetChannelInfo(ctx context.Context, channelID string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.api == nil {
		return "", fmt.Errorf("client is not connected")
	}

	c.logger.Debug().Str("channel_id", channelID).Msg("fetching channel info")

	// TODO: Implement actual channel info fetching logic using c.api

	return "", fmt.Errorf("not implemented")
}

// Authenticate performs phone authentication
// This is a helper method for initial setup, not part of the domain.TelegramClient interface
func (c *MTProtoClient) Authenticate(ctx context.Context, phoneNumber string) error {
	if c.client == nil {
		return fmt.Errorf("client not initialized, call Connect first")
	}

	flow := auth.NewFlow(
		auth.Constant(phoneNumber, "", auth.CodeAuthenticatorFunc(
			func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
				// TODO: Implement code input mechanism
				// This could be via console, file, or API endpoint
				return "", fmt.Errorf("code input not implemented")
			},
		)),
		auth.SendCodeOptions{},
	)

	return c.client.Auth().IfNecessary(ctx, flow)
}

// Ensure MTProtoClient implements domain.TelegramClient interface
var _ domain.TelegramClient = (*MTProtoClient)(nil)
