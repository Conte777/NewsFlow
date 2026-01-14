package domain

import (
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"domain",
	subscription.Module,
)
