package logger

import (
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides logger for fx DI
var Module = fx.Module("logger",
	fx.Provide(NewLogger),
)

// NewLogger creates a new logger from config
func NewLogger(cfg *config.LoggingConfig) zerolog.Logger {
	return New(cfg.Level)
}
