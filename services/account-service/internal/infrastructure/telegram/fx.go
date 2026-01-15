package telegram

import (
	"context"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides Telegram account manager for fx DI
var Module = fx.Module("telegram",
	fx.Provide(NewAccountRepositoryFx),
	fx.Provide(NewAccountManagerFx),
)

// NewAccountRepositoryFx creates an AccountRepository for fx DI
func NewAccountRepositoryFx(db *gorm.DB) *AccountRepository {
	return NewAccountRepository(db)
}

// NewAccountManagerFx creates an account manager with lifecycle hooks for fx DI
func NewAccountManagerFx(
	lc fx.Lifecycle,
	telegramCfg *config.TelegramConfig,
	db *gorm.DB,
	accountRepo *AccountRepository,
	logger zerolog.Logger,
) (domain.AccountManager, error) {
	manager := NewAccountManager().(*accountManager)
	manager.WithLogger(logger)

	// Set client factory to use PostgreSQL session storage
	manager.clientFactory = func(cfg MTProtoClientConfig) (domain.TelegramClient, error) {
		return NewMTProtoClientWithDB(cfg, db)
	}

	// Channel to stop the sync goroutine
	stopSync := make(chan struct{})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Load phone numbers from database
			phoneNumbers, err := accountRepo.GetEnabledPhoneNumbers(ctx)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to load accounts from database")
				return err
			}

			if len(phoneNumbers) == 0 {
				logger.Warn().Msg("No enabled accounts found in database")
			}

			// Initialize accounts from database
			report := manager.InitializeAccounts(ctx, domain.AccountInitConfig{
				APIID:         telegramCfg.APIID,
				APIHash:       telegramCfg.APIHash,
				SessionDir:    telegramCfg.SessionDir,
				Accounts:      phoneNumbers,
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
				Msg("Telegram accounts initialized from database")

			// Start periodic sync goroutine
			go runAccountSync(manager, accountRepo, telegramCfg, logger, stopSync)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop sync goroutine
			close(stopSync)

			// Shutdown accounts
			disconnected := manager.Shutdown(ctx)
			logger.Info().
				Int("disconnected", disconnected).
				Msg("Telegram accounts disconnected")
			return nil
		},
	})

	return manager, nil
}

// runAccountSync periodically checks for new accounts in the database
func runAccountSync(
	manager *accountManager,
	repo *AccountRepository,
	cfg *config.TelegramConfig,
	logger zerolog.Logger,
	stopCh <-chan struct{},
) {
	ticker := time.NewTicker(cfg.AccountSyncInterval)
	defer ticker.Stop()

	logger.Info().
		Dur("interval", cfg.AccountSyncInterval).
		Msg("Account sync started")

	for {
		select {
		case <-stopCh:
			logger.Info().Msg("Account sync stopped")
			return
		case <-ticker.C:
			syncAccounts(manager, repo, cfg, logger)
		}
	}
}

// syncAccounts checks for new accounts and initializes them
func syncAccounts(
	manager *accountManager,
	repo *AccountRepository,
	cfg *config.TelegramConfig,
	logger zerolog.Logger,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	phoneNumbers, err := repo.GetEnabledPhoneNumbers(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load accounts from database during sync")
		return
	}

	report := manager.SyncAccounts(ctx, domain.AccountInitConfig{
		APIID:         cfg.APIID,
		APIHash:       cfg.APIHash,
		SessionDir:    cfg.SessionDir,
		Accounts:      phoneNumbers,
		Logger:        logger,
		MaxConcurrent: 10,
	})

	if report.TotalAccounts > 0 {
		logger.Info().
			Int("new_accounts", report.TotalAccounts).
			Int("successful", report.SuccessfulAccounts).
			Int("failed", report.FailedAccounts).
			Msg("Account sync completed")
	}
}
