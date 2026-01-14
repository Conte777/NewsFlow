package metrics

import "go.uber.org/fx"

// Module provides metrics for fx DI
var Module = fx.Module("metrics",
	fx.Provide(func() *Metrics {
		return GetDefaultMetrics()
	}),
)
