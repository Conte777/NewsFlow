package cache

import (
	"context"
	"sync"

	channeldeps "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/rs/zerolog"
)

// messageIDCache implements in-memory cache for LastProcessedMessageID
// to prevent race conditions between real-time handler and fallback collector
type messageIDCache struct {
	data        map[string]int
	mu          sync.RWMutex
	channelRepo channeldeps.ChannelRepository
	logger      zerolog.Logger
}

// NewMessageIDCache creates a new MessageIDCache instance
func NewMessageIDCache(
	channelRepo channeldeps.ChannelRepository,
	logger zerolog.Logger,
) channeldeps.MessageIDCache {
	return &messageIDCache{
		data:        make(map[string]int),
		channelRepo: channelRepo,
		logger:      logger.With().Str("component", "message_id_cache").Logger(),
	}
}

// Get returns the cached lastProcessedMessageID for a channel
func (c *messageIDCache) Get(channelID string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	messageID, exists := c.data[channelID]
	return messageID, exists
}

// SetIfGreater atomically updates the cache only if newMessageID > current
func (c *messageIDCache) SetIfGreater(channelID string, newMessageID int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, exists := c.data[channelID]
	if !exists || newMessageID > current {
		c.data[channelID] = newMessageID
		c.logger.Debug().
			Str("channel_id", channelID).
			Int("old_message_id", current).
			Int("new_message_id", newMessageID).
			Msg("updated cached message ID")
		return true
	}

	return false
}

// Delete removes a channel from the cache
func (c *messageIDCache) Delete(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, channelID)
	c.logger.Debug().
		Str("channel_id", channelID).
		Msg("deleted channel from cache")
}

// LoadFromDB loads all channel message IDs from the database into the cache
func (c *messageIDCache) LoadFromDB(ctx context.Context) error {
	channels, err := c.channelRepo.GetAllChannels(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, channel := range channels {
		c.data[channel.ChannelID] = channel.LastProcessedMessageID
	}

	c.logger.Info().
		Int("channels_loaded", len(channels)).
		Msg("loaded message IDs from database")

	return nil
}
