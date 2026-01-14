package workers

import (
	"context"

	"go.uber.org/fx"
)

// Module provides news workers for fx DI
var Module = fx.Module("news-workers",
	fx.Provide(NewCollectorWorker),
	fx.Invoke(registerLifecycle),
)

// registerLifecycle registers collector worker with fx.Lifecycle
func registerLifecycle(lc fx.Lifecycle, w *CollectorWorker) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			w.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			w.Stop()
			return nil
		},
	})
}
