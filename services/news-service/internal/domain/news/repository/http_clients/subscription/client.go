package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/yourusername/telegram-news-feed/news-service/config"
	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news/deps"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

type subscribersResponse struct {
	UserIDs []int64 `json:"user_ids"`
}

func NewClient(cfg *config.SubscriptionServiceConfig, logger zerolog.Logger) deps.SubscriptionClient {
	client := &Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}

	logger.Info().
		Str("base_url", cfg.URL).
		Msg("Subscription client initialized")

	return client
}

func (c *Client) GetChannelSubscribers(ctx context.Context, channelID string) ([]int64, error) {
	url := fmt.Sprintf("%s/api/v1/channels/%s/subscribers", c.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to get channel subscribers, returning empty list")
		return []int64{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Debug().
			Str("channel_id", channelID).
			Msg("No subscribers found for channel")
		return []int64{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn().
			Int("status_code", resp.StatusCode).
			Str("channel_id", channelID).
			Msg("Unexpected status code from subscription service")
		return []int64{}, nil
	}

	var result subscribersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Msg("Failed to decode response, returning empty list")
		return []int64{}, nil
	}

	return result.UserIDs, nil
}
