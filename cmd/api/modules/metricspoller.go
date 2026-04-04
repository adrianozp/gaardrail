package modules

import (
	"context"

	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/repositories/readers"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/internal/controller"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func MetricsPollerFactories() fx.Option {
	return fx.Provide(
		controller.New,
		processmetrics.NewProcessMetricsUseCase,
		readers.NewMetricsReader,
		pollmetrics.New,
	)
}

func MetricsPollerInjections() fx.Option {
	return fx.Provide(
		func(c *controller.Controller) processmetrics.Controller { return c },
		func(uc processmetrics.ProcessMetricsUseCase) pollmetrics.ProcessMetrics { return uc },
		func(r readers.MetricsReader) pollmetrics.MetricsReader { return r },
	)
}

func MetricsPollerLifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, ph *pollmetrics.PollingHandler, cfg config.Config) {
		if !cfg.MetricsPoller.Enabled {
			return
		}
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return ph.Start(ctx)
			},
		})
	})
}
