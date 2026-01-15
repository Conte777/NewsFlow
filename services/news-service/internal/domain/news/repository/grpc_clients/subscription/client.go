package subscription

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Conte777/NewsFlow/pkg/proto/subscription/v1"
	"github.com/yourusername/telegram-news-feed/news-service/config"
	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/deps"
)

type Client struct {
	client pb.SubscriptionServiceClient
	conn   *grpc.ClientConn
	logger zerolog.Logger
}

func NewClient(cfg *config.SubscriptionServiceConfig, logger zerolog.Logger) (deps.SubscriptionClient, error) {
	conn, err := grpc.NewClient(cfg.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error().Err(err).
			Str("address", cfg.GRPCAddr).
			Msg("Failed to connect to subscription service")
		return nil, err
	}

	logger.Info().
		Str("address", cfg.GRPCAddr).
		Msg("Connected to subscription service via gRPC")

	return &Client{
		client: pb.NewSubscriptionServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *Client) GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c.logger.Debug().
		Str("channel_id", channelID).
		Msg("Fetching channel subscribers via gRPC")

	resp, err := c.client.GetChannelSubscribers(ctx, &pb.GetChannelSubscribersRequest{
		ChannelId: channelID,
	})
	if err != nil {
		c.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Msg("gRPC GetChannelSubscribers failed, returning empty list")
		return []int64{}, nil
	}

	c.logger.Debug().
		Str("channel_id", channelID).
		Int("count", len(resp.UserIds)).
		Msg("Received channel subscribers")

	return resp.UserIds, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
