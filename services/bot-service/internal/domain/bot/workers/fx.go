// Package workers contains background workers for the bot domain
package workers

import (
	"context"

	"go.uber.org/fx"
)

// Module provides workers for fx dependency injection
var Module = fx.Module("bot-workers",
	fx.Provide(NewNewsConsumer),
	fx.Invoke(registerNewsConsumerLifecycle),
)

// registerNewsConsumerLifecycle registers news consumer lifecycle hooks
func registerNewsConsumerLifecycle(lc fx.Lifecycle, consumer *NewsConsumer) {
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

// registerRejectionConsumerLifecycle registers rejection consumer lifecycle hooks
// Called from bot/fx.go after TelegramSender is wired
func registerRejectionConsumerLifecycle(lc fx.Lifecycle, consumer *RejectionConsumer) {
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
