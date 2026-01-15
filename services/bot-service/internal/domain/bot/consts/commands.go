// Package consts contains constants for the bot domain
package consts

// Command represents a bot command
type Command struct {
	Name        string
	Description string
}

// Bot commands
var (
	CommandStart       = Command{Name: "start", Description: "Start the bot"}
	CommandHelp        = Command{Name: "help", Description: "Show help message"}
	CommandSubscribe   = Command{Name: "subscribe", Description: "Subscribe to channels"}
	CommandUnsubscribe = Command{Name: "unsubscribe", Description: "Unsubscribe from channels"}
	CommandList        = Command{Name: "list", Description: "List your subscriptions"}
)

// AllCommands contains all available bot commands for menu registration
var AllCommands = []Command{
	CommandStart,
	CommandHelp,
	CommandSubscribe,
	CommandUnsubscribe,
	CommandList,
}
