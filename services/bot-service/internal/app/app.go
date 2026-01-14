// Package app contains application bootstrap
package app

import (
	"go.uber.org/fx"

	"github.com/Conte777/newsflow/services/bot-service/config"
	"github.com/Conte777/newsflow/services/bot-service/internal/domain"
	"github.com/Conte777/newsflow/services/bot-service/internal/infrastructure"
)

// CreateApp creates fx application with all modules
func CreateApp() fx.Option {
	return fx.Options(
		// Configuration
		fx.Provide(config.Out),

		// Infrastructure (logger, telegram bot)
		infrastructure.Module,

		// Domain (bot business logic)
		domain.Module,
	)
}
