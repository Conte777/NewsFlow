package logger

import (
	"github.com/Conte777/NewsFlow/services/subscription-service/config"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"logger",
	fx.Provide(NewLogger),
)

func NewLogger(cfg *config.LoggingConfig) zerolog.Logger {
	return New(cfg.Level)
}
