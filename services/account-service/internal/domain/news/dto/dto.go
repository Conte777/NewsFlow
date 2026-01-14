package dto

import "time"

// NewsItemDTO represents a news item for Kafka message
type NewsItemDTO struct {
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	MessageID   int       `json:"message_id"`
	Content     string    `json:"content"`
	MediaURLs   []string  `json:"media_urls"`
	Date        time.Time `json:"date"`
}

// CollectNewsResponse represents the response for news collection
type CollectNewsResponse struct {
	TotalCollected int `json:"totalCollected"`
	TotalSent      int `json:"totalSent"`
	TotalSkipped   int `json:"totalSkipped"`
	ChannelsCount  int `json:"channelsCount"`
}
