// Package domain contains all domain modules
package domain

import (
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/domain/bot"
)

// Module aggregates all domain modules for fx dependency injection
var Module = fx.Module("domain",
	bot.Module,
)
