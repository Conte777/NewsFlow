package cache

import (
	"container/list"
	"sync"

	"github.com/rs/zerolog"
)

// RecentMessagesCache stores recent message IDs per channel for deletion checking
// Uses LRU-like structure to keep last N message IDs per channel
type RecentMessagesCache struct {
	data       map[string]*channelMessages
	mu         sync.RWMutex
	maxPerChan int
	logger     zerolog.Logger
}

// channelMessages holds recent message IDs for a single channel
type channelMessages struct {
	ids  *list.List          // Ordered list of message IDs (newest at front)
	set  map[int]*list.Element // Fast lookup for deduplication
}

// NewRecentMessagesCache creates a new RecentMessagesCache instance
func NewRecentMessagesCache(maxPerChannel int, logger zerolog.Logger) *RecentMessagesCache {
	if maxPerChannel <= 0 {
		maxPerChannel = 100 // Default
	}
	return &RecentMessagesCache{
		data:       make(map[string]*channelMessages),
		maxPerChan: maxPerChannel,
		logger:     logger.With().Str("component", "recent_messages_cache").Logger(),
	}
}

// Add adds a message ID to the cache for a channel
// Automatically evicts oldest entries when limit is reached
func (c *RecentMessagesCache) Add(channelID string, messageID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cm, exists := c.data[channelID]
	if !exists {
		cm = &channelMessages{
			ids: list.New(),
			set: make(map[int]*list.Element),
		}
		c.data[channelID] = cm
	}

	// Check if already exists
	if _, found := cm.set[messageID]; found {
		return
	}

	// Add to front (newest)
	elem := cm.ids.PushFront(messageID)
	cm.set[messageID] = elem

	// Evict oldest if over limit
	for cm.ids.Len() > c.maxPerChan {
		oldest := cm.ids.Back()
		if oldest != nil {
			oldID := oldest.Value.(int)
			delete(cm.set, oldID)
			cm.ids.Remove(oldest)
		}
	}
}

// GetRecent returns the most recent N message IDs for a channel
// Returns IDs in descending order (newest first)
func (c *RecentMessagesCache) GetRecent(channelID string, count int) []int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cm, exists := c.data[channelID]
	if !exists || cm.ids.Len() == 0 {
		return nil
	}

	result := make([]int, 0, min(count, cm.ids.Len()))
	for e := cm.ids.Front(); e != nil && len(result) < count; e = e.Next() {
		result = append(result, e.Value.(int))
	}

	return result
}

// Remove removes specific message IDs from the cache for a channel
func (c *RecentMessagesCache) Remove(channelID string, messageIDs []int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cm, exists := c.data[channelID]
	if !exists {
		return
	}

	for _, msgID := range messageIDs {
		if elem, found := cm.set[msgID]; found {
			cm.ids.Remove(elem)
			delete(cm.set, msgID)
		}
	}

	c.logger.Debug().
		Str("channel_id", channelID).
		Ints("message_ids", messageIDs).
		Int("remaining", cm.ids.Len()).
		Msg("removed message IDs from recent cache")
}

// GetAllChannelIDs returns all channel IDs that have cached messages
func (c *RecentMessagesCache) GetAllChannelIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]string, 0, len(c.data))
	for channelID := range c.data {
		result = append(result, channelID)
	}
	return result
}

// Clear removes all entries for a channel
func (c *RecentMessagesCache) Clear(channelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, channelID)
}

// Count returns the number of cached message IDs for a channel
func (c *RecentMessagesCache) Count(channelID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cm, exists := c.data[channelID]
	if !exists {
		return 0
	}
	return cm.ids.Len()
}
