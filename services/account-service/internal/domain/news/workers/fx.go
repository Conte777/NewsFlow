package workers

import (
	"context"

	"go.uber.org/fx"
)

// Module provides news workers for fx DI
var Module = fx.Module("news-workers",
	fx.Provide(NewCollectorWorker),
	fx.Provide(NewDeletionChecker),
	fx.Invoke(registerCollectorLifecycle),
	fx.Invoke(registerDeletionCheckerLifecycle),
)

// registerCollectorLifecycle registers collector worker with fx.Lifecycle
func registerCollectorLifecycle(lc fx.Lifecycle, w *CollectorWorker) {
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

// registerDeletionCheckerLifecycle registers deletion checker worker with fx.Lifecycle
func registerDeletionCheckerLifecycle(lc fx.Lifecycle, w *DeletionChecker) {
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
