package deps

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/account/entities"
)

// TelegramClient defines interface for Telegram MTProto operations
// Note: GetChannelMessages and GetChannelInfo use domain.NewsItem and domain.ChannelInfo
// for backward compatibility until full migration is complete
type TelegramClient interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	JoinChannel(ctx context.Context, channelID string) error
	LeaveChannel(ctx context.Context, channelID string) error
	IsConnected() bool
	GetAccountID() string
}

// AccountManager manages multiple Telegram accounts
type AccountManager interface {
	GetAvailableAccount() (TelegramClient, error)
	GetAllAccounts() []TelegramClient
	AddAccount(client TelegramClient) error
	RemoveAccount(accountID string) error
	InitializeAccounts(ctx context.Context, cfg entities.AccountInitConfig) *entities.InitializationReport
	Shutdown(ctx context.Context) int
	GetActiveAccountCount() int
}
