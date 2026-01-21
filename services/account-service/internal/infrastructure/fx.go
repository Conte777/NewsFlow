package infrastructure

import (
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/database"
	httpfx "github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/http"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/kafka"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/logger"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/metrics"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/telegram"
	"go.uber.org/fx"
)

// Module aggregates all infrastructure modules
// Note: handlers.Module is loaded separately in app.go after cache.Module (handlers depends on MessageIDCache)
var Module = fx.Module("infrastructure",
	logger.Module,
	database.Module, // Must be before telegram (telegram depends on *gorm.DB)
	metrics.Module,
	kafka.Module,
	telegram.Module,
	httpfx.Module,
)
