package news

import (
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news/usecase/business"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news/workers"
	"go.uber.org/fx"
)

// Module provides news domain components for fx DI
var Module = fx.Module("news",
	fx.Provide(business.NewUseCase),
	workers.Module,
)
