package domain

import (
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/news-service/internal/domain/news"
)

// Module aggregates all domain modules
var Module = fx.Module(
	"domain",
	news.Module,
)
