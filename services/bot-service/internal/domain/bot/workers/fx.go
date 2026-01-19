// Package workers contains background workers for the bot domain
package workers

import (
	"context"

	"go.uber.org/fx"
)

// Module provides workers for fx dependency injection
var Module = fx.Module("bot-workers",
	fx.Provide(NewNewsConsumer),
	fx.Provide(NewDeleteConsumer),
	fx.Provide(NewEditConsumer),
	fx.Invoke(registerNewsConsumerLifecycle),
	fx.Invoke(registerDeleteConsumerLifecycle),
	fx.Invoke(registerEditConsumerLifecycle),
)

// registerNewsConsumerLifecycle registers news consumer lifecycle hooks
func registerNewsConsumerLifecycle(lc fx.Lifecycle, consumer *NewsConsumer) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return consumer.Stop()
		},
	})
}

// registerDeleteConsumerLifecycle registers delete consumer lifecycle hooks
func registerDeleteConsumerLifecycle(lc fx.Lifecycle, consumer *DeleteConsumer) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return consumer.Stop()
		},
	})
}

// registerEditConsumerLifecycle registers edit consumer lifecycle hooks
func registerEditConsumerLifecycle(lc fx.Lifecycle, consumer *EditConsumer) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			consumer.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return consumer.Stop()
		},
	})
}
