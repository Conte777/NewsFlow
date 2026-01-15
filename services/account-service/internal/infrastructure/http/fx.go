package http

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/http/server"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides HTTP server for fx DI
var Module = fx.Module("http",
	fx.Provide(NewServerFx),
)

// NewServerFx creates HTTP server with lifecycle hooks for fx DI
func NewServerFx(
	lc fx.Lifecycle,
	serviceCfg *config.ServiceConfig,
	logger zerolog.Logger,
) *server.Server {
	srv := server.NewServer(serviceCfg.Port, logger)

	// Register Prometheus metrics endpoint
	srv.RegisterMetrics()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return srv.Start()
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
