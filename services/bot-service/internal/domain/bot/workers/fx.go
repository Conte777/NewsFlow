// Package workers contains background workers for the bot domain
package workers

import (
	"context"

	"go.uber.org/fx"
)

// Module provides workers for fx dependency injection
var Module = fx.Module("bot-workers",
	fx.Provide(NewNewsConsumer),
	fx.Invoke(registerLifecycle),
)

// registerLifecycle registers worker lifecycle hooks
func registerLifecycle(lc fx.Lifecycle, consumer *NewsConsumer) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			consumer.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return consumer.Stop()
		},
	})
}
