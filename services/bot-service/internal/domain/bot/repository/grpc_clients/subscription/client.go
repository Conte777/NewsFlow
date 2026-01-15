package subscription

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Conte777/NewsFlow/pkg/proto/subscription/v1"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot/entities"
)

// Client implements deps.SubscriptionRepository using gRPC
type Client struct {
	client pb.SubscriptionServiceClient
	conn   *grpc.ClientConn
	logger zerolog.Logger
}

// NewClient creates a new gRPC subscription client
func NewClient(address string, logger zerolog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error().Err(err).Str("address", address).Msg("failed to connect to subscription service")
		return nil, err
	}

	logger.Info().Str("address", address).Msg("connected to subscription service via gRPC")

	return &Client{
		client: pb.NewSubscriptionServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

// GetUserSubscriptions returns list of user subscriptions via gRPC
func (c *Client) GetUserSubscriptions(ctx context.Context, userID int64) ([]entities.Subscription, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c.logger.Debug().Int64("user_id", userID).Msg("fetching user subscriptions via gRPC")

	resp, err := c.client.GetUserSubscriptions(ctx, &pb.GetUserSubscriptionsRequest{
		UserId: userID,
	})
	if err != nil {
		c.logger.Error().Err(err).Int64("user_id", userID).Msg("gRPC GetUserSubscriptions failed")
		return nil, err
	}

	result := make([]entities.Subscription, len(resp.Subscriptions))
	for i, sub := range resp.Subscriptions {
		result[i] = entities.Subscription{
			UserID:      sub.UserId,
			ChannelID:   sub.ChannelId,
			ChannelName: sub.ChannelName,
		}
	}

	c.logger.Debug().Int64("user_id", userID).Int("count", len(result)).Msg("received user subscriptions")

	return result, nil
}

// SaveSubscription is not implemented - subscriptions are created via Kafka
func (c *Client) SaveSubscription(ctx context.Context, subscription *entities.Subscription) error {
	return nil
}

// DeleteSubscription is not implemented - subscriptions are deleted via Kafka
func (c *Client) DeleteSubscription(ctx context.Context, userID int64, channelID string) error {
	return nil
}

// CheckSubscription checks if user is subscribed to a channel
func (c *Client) CheckSubscription(ctx context.Context, userID int64, channelID string) (bool, error) {
	subs, err := c.GetUserSubscriptions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, sub := range subs {
		if sub.ChannelID == channelID {
			return true, nil
		}
	}

	return false, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
