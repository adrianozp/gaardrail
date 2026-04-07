package modules

import (
	"context"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	promrecorder "github.com/adrianozp/gaardrail/internal/metrics/prometheus"
	"go.uber.org/fx"
)

// MetricsLifecycle wires the global metrics recorder into the fx lifecycle.
// It sets up an AsyncRecorder backed by Prometheus on startup and drains the
// channel on graceful shutdown.
func MetricsLifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle) {
		prom := promrecorder.Recorder{}
		async, stop := metrics.NewAsync(prom, 256)
		metrics.SetRecorder(async)
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				stop()
				return nil
			},
		})
	})
}
