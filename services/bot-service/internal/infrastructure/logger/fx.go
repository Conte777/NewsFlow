// Package logger contains logger infrastructure
package logger

import (
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"github.com/Conte777/newsflow/services/bot-service/config"
)

// Module provides logger for fx dependency injection
var Module = fx.Module("logger",
	fx.Provide(provideLogger),
)

// provideLogger creates logger from config
func provideLogger(cfg *config.LoggingConfig) zerolog.Logger {
	return New(cfg.Level)
}
