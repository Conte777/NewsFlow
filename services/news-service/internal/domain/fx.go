package domain

import (
	"go.uber.org/fx"

	"github.com/yourusername/telegram-news-feed/news-service/internal/domain/news"
)

// Module aggregates all domain modules
var Module = fx.Module(
	"domain",
	news.Module,
)
