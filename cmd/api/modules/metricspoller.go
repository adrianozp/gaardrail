package modules

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/app/repositories/readers"
	cloudwatchrepo "github.com/adrianozp/gaardrail/app/repositories/readers/cloudwatch"
	jsonmetricsrepo "github.com/adrianozp/gaardrail/app/repositories/readers/jsonmetrics"
	prometheusrepo "github.com/adrianozp/gaardrail/app/repositories/readers/prometheus"
	prometheusapirepo "github.com/adrianozp/gaardrail/app/repositories/readers/prometheusapi"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func MetricsPollerFactories() fx.Option {
	return fx.Provide(
		processmetrics.NewProcessMetricsUseCase,
		newMetricsReader,
		pollmetrics.New,
	)
}

func newMetricsReader(cfg config.Config) (readers.MetricsReader, error) {
	if !cfg.MetricsPoller.Enabled {
		return readers.Noop{}, nil
	}
	switch cfg.MetricsPoller.Protocol {
	case "prometheus":
		return prometheusrepo.New(cfg), nil
	case "json":
		return jsonmetricsrepo.New(cfg), nil
	case "prometheusapi":
		return prometheusapirepo.New(cfg), nil
	case "cloudwatch":
		return cloudwatchrepo.New(cfg)
	default:
		return nil, fmt.Errorf("metricspoller: unknown protocol %q", cfg.MetricsPoller.Protocol)
	}
}

func MetricsPollerInjections() fx.Option {
	return fx.Provide(
		func(c controllers.Controller) processmetrics.Controller { return c },
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
				return ph.Start(context.Background())
			},
		})
	})
}
