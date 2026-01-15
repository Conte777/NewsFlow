package dto

import "time"

// NewsReceivedEvent represents incoming news event from account service
type NewsReceivedEvent struct {
	ChannelID   string    `json:"channel_id" validate:"required"`
	ChannelName string    `json:"channel_name" validate:"required"`
	MessageID   int       `json:"message_id" validate:"required"`
	Content     string    `json:"content"`
	MediaURLs   []string  `json:"media_urls"`
	Date        time.Time `json:"date"`
}

// NewsDeliveryEvent represents outgoing news delivery event to bot service
type NewsDeliveryEvent struct {
	NewsID      uint     `json:"news_id"`
	UserID      int64    `json:"user_id"`
	ChannelID   string   `json:"channel_id"`
	ChannelName string   `json:"channel_name"`
	Content     string   `json:"content"`
	MediaURLs   []string `json:"media_urls"`
	Timestamp   int64    `json:"timestamp"`
}

// GetUserNewsRequest represents request to get user's news history
type GetUserNewsRequest struct {
	UserID int64 `query:"userId" validate:"required"`
	Limit  int   `query:"limit" validate:"min=1,max=100"`
}

// GetUserNewsResponse represents response with user's news history
type GetUserNewsResponse struct {
	News []NewsItem `json:"news"`
}

// NewsItem represents a single news item in response
type NewsItem struct {
	ID          uint      `json:"id"`
	ChannelID   string    `json:"channelId"`
	ChannelName string    `json:"channelName"`
	Content     string    `json:"content"`
	MediaURLs   []string  `json:"mediaUrls"`
	DeliveredAt time.Time `json:"deliveredAt"`
}

// ProcessNewsRequest represents request to process news
type ProcessNewsRequest struct {
	ChannelID   string   `json:"channelId" validate:"required"`
	ChannelName string   `json:"channelName" validate:"required"`
	MessageID   int      `json:"messageId" validate:"required"`
	Content     string   `json:"content"`
	MediaURLs   []string `json:"mediaUrls"`
}

// ProcessNewsResponse represents response after processing news
type ProcessNewsResponse struct {
	NewsID uint `json:"newsId"`
}
