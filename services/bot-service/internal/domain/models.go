package domain

import "time"

// User represents a Telegram user
type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	CreatedAt time.Time
}

// Subscription represents a user subscription to a channel
type Subscription struct {
	UserID      int64
	ChannelID   string
	ChannelName string
	CreatedAt   time.Time
}

// NewsMessage represents a news message to be delivered
type NewsMessage struct {
	ID          string
	UserID      int64
	ChannelID   string
	ChannelName string
	Content     string
	MediaURLs   []string
	MessageID   int
	CreatedAt   time.Time
}

// Command represents a bot command
type Command struct {
	Name        string
	Description string
}

// Common commands
var (
	CommandStart      = Command{Name: "start", Description: "Start the bot"}
	CommandHelp       = Command{Name: "help", Description: "Show help message"}
	CommandSubscribe  = Command{Name: "subscribe", Description: "Subscribe to a channel"}
	CommandUnsubscribe = Command{Name: "unsubscribe", Description: "Unsubscribe from a channel"}
	CommandList       = Command{Name: "list", Description: "List your subscriptions"}
)
