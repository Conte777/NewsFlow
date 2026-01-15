// Package telegram contains Telegram bot infrastructure
package telegram

import (
	"context"

	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/bot-service/config"
)

// Module provides Telegram bot for fx dependency injection
var Module = fx.Module("telegram",
	fx.Provide(provideBot),
	fx.Invoke(registerLifecycle),
)

// provideBot creates Telegram bot from config
func provideBot(cfg *config.TelegramConfig, logger zerolog.Logger) (*Bot, error) {
	return NewBot(cfg.BotToken, logger)
}

// registerLifecycle registers bot lifecycle hooks
func registerLifecycle(lc fx.Lifecycle, bot *Bot) {
	var cancel context.CancelFunc

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// Create a long-lived context for the bot
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())

			// Start bot in a goroutine since it's a blocking call
			go func() {
				_ = bot.Start(ctx)
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			if cancel != nil {
				cancel()
			}
			return bot.Stop()
		},
	})
}
