package infrastructure

import (
	httpfx "github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/http"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/kafka"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/logger"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/metrics"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/telegram"
	"go.uber.org/fx"
)

// Module aggregates all infrastructure modules
var Module = fx.Module("infrastructure",
	logger.Module,
	metrics.Module,
	telegram.Module,
	kafka.Module,
	httpfx.Module,
)
