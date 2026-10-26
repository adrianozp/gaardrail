package modules

import (
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	promrecorder "github.com/adrianozp/gaardrail/internal/metrics/prometheus"
	"go.uber.org/fx"
)

func MetricsLifecycle() fx.Option {
	return fx.Invoke(startAsyncRecorder)
}

func startAsyncRecorder(lc fx.Lifecycle) {
	async, stop := metrics.NewAsync(promrecorder.Recorder{}, 256)
	metrics.SetRecorder(async)
	lc.Append(fx.StopHook(stop))
}
