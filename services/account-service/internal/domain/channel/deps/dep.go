package deps

import (
	"context"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/channel/entities"
)

// ChannelRepository defines interface for channel subscription storage
type ChannelRepository interface {
	AddChannel(ctx context.Context, channelID, channelName string) error
	RemoveChannel(ctx context.Context, channelID string) error
	GetAllChannels(ctx context.Context) ([]entities.ChannelSubscription, error)
	GetChannel(ctx context.Context, channelID string) (*entities.ChannelSubscription, error)
	ChannelExists(ctx context.Context, channelID string) (bool, error)
	UpdateLastProcessedMessageID(ctx context.Context, channelID string, messageID int) error
}

// SubscriptionEventHandler handles subscription events
type SubscriptionEventHandler interface {
	HandleSubscriptionCreated(ctx context.Context, userID int64, channelID, channelName string) error
	HandleSubscriptionDeleted(ctx context.Context, userID int64, channelID string) error
}
