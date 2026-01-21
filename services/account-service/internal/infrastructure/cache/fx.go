package cache

import (
	"context"

	channeldeps "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides cache components for fx DI
var Module = fx.Module("cache",
	fx.Provide(NewMessageIDCache),
	fx.Invoke(registerCacheLifecycle),
)

func registerCacheLifecycle(
	lc fx.Lifecycle,
	cache channeldeps.MessageIDCache,
	logger zerolog.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info().Msg("loading message ID cache from database")
			if err := cache.LoadFromDB(ctx); err != nil {
				logger.Error().Err(err).Msg("failed to load message ID cache from database")
				return err
			}
			return nil
		},
	})
}
