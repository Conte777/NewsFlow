// Package infrastructure contains infrastructure layer components
package infrastructure

import (
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/infrastructure/logger"
	"github.com/Conte777/NewsFlow/services/bot-service/internal/infrastructure/telegram"
)

// Module provides all infrastructure components for fx dependency injection
var Module = fx.Module("infrastructure",
	logger.Module,
	telegram.Module,
)
