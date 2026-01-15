package memory

import (
	"context"
	"sync"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/entities"
	channelerrors "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/errors"
)

// channelRepository implements deps.ChannelRepository using in-memory storage
type channelRepository struct {
	mu       sync.RWMutex
	channels map[string]*entities.ChannelSubscription
}

// NewRepository creates a new in-memory channel repository
func NewRepository() deps.ChannelRepository {
	return &channelRepository{
		channels: make(map[string]*entities.ChannelSubscription),
	}
}

// AddChannel adds a channel subscription
func (r *channelRepository) AddChannel(ctx context.Context, channelID, channelName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[channelID] = &entities.ChannelSubscription{
		ChannelID:   channelID,
		ChannelName: channelName,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	return nil
}

// RemoveChannel removes a channel subscription
func (r *channelRepository) RemoveChannel(ctx context.Context, channelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.channels[channelID]; !exists {
		return channelerrors.ErrChannelNotFound
	}

	delete(r.channels, channelID)
	return nil
}

// GetAllChannels retrieves all subscribed channels
func (r *channelRepository) GetAllChannels(ctx context.Context) ([]entities.ChannelSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	channels := make([]entities.ChannelSubscription, 0, len(r.channels))
	for _, channel := range r.channels {
		if channel.IsActive {
			channels = append(channels, *channel)
		}
	}

	return channels, nil
}

// GetChannel retrieves a specific channel
func (r *channelRepository) GetChannel(ctx context.Context, channelID string) (*entities.ChannelSubscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	channel, exists := r.channels[channelID]
	if !exists {
		return nil, channelerrors.ErrChannelNotFound
	}

	return channel, nil
}

// ChannelExists checks if channel exists
func (r *channelRepository) ChannelExists(ctx context.Context, channelID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.channels[channelID]
	return exists, nil
}

// UpdateLastProcessedMessageID updates the last processed message ID for a channel
func (r *channelRepository) UpdateLastProcessedMessageID(ctx context.Context, channelID string, messageID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	channel, exists := r.channels[channelID]
	if !exists {
		return channelerrors.ErrChannelNotFound
	}

	channel.LastProcessedMessageID = messageID
	return nil
}
