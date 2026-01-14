package entities

import "time"

// NewsItem represents a news message from a channel
type NewsItem struct {
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	MessageID   int       `json:"message_id"`
	Content     string    `json:"content"`
	MediaURLs   []string  `json:"media_urls"`
	Date        time.Time `json:"date"`
}
