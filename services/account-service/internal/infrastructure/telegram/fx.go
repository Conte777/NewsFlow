package telegram

import (
	"context"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides Telegram account manager for fx DI
var Module = fx.Module("telegram",
	fx.Provide(NewAccountManagerFx),
)

// NewAccountManagerFx creates an account manager with lifecycle hooks for fx DI
func NewAccountManagerFx(
	lc fx.Lifecycle,
	telegramCfg *config.TelegramConfig,
	logger zerolog.Logger,
) (domain.AccountManager, error) {
	manager := NewAccountManager().(*accountManager)
	manager.WithLogger(logger)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			report := manager.InitializeAccounts(ctx, domain.AccountInitConfig{
				APIID:         telegramCfg.APIID,
				APIHash:       telegramCfg.APIHash,
				SessionDir:    telegramCfg.SessionDir,
				Accounts:      telegramCfg.Accounts,
				Logger:        logger,
				MaxConcurrent: 10,
			})

			if report.SuccessfulAccounts < telegramCfg.MinRequiredAccounts {
				logger.Error().
					Int("successful", report.SuccessfulAccounts).
					Int("required", telegramCfg.MinRequiredAccounts).
					Msg("Not enough accounts initialized")
			}

			logger.Info().
				Int("successful", report.SuccessfulAccounts).
				Int("failed", report.FailedAccounts).
				Int("total", report.TotalAccounts).
				Msg("Telegram accounts initialized")

			return nil
		},
		OnStop: func(ctx context.Context) error {
			disconnected := manager.Shutdown(ctx)
			logger.Info().
				Int("disconnected", disconnected).
				Msg("Telegram accounts disconnected")
			return nil
		},
	})

	return manager, nil
}
