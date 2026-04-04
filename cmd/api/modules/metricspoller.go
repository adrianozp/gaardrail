package modules

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/repositories"
	jsonmetricsrepo "github.com/adrianozp/gaardrail/app/repositories/jsonmetrics"
	prometheusrepo "github.com/adrianozp/gaardrail/app/repositories/prometheus"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/internal/controller"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func MetricsPollerFactories() fx.Option {
	return fx.Provide(
		controller.New,
		processmetrics.NewProcessMetricsUseCase,
		newMetricsReader,
		pollmetrics.New,
	)
}

func MetricsPollerInjections() fx.Option {
	return fx.Provide(
		func(c *controller.Controller) processmetrics.Controller { return c },
		func(uc processmetrics.ProcessMetricsUseCase) pollmetrics.ProcessMetrics { return uc },
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

func newMetricsReader(cfg config.Config) (repositories.MetricsReader, error) {
	if !cfg.MetricsPoller.Enabled {
		return &noopMetricsReader{}, nil
	}
	switch cfg.MetricsPoller.Protocol {
	case "prometheus":
		return prometheusrepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	case "json":
		return jsonmetricsrepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	default:
		return nil, fmt.Errorf("metricspoller: unknown protocol %q", cfg.MetricsPoller.Protocol)
	}
}

// noopMetricsReader is used when MetricsPoller.Enabled is false.
type noopMetricsReader struct{}

func (n *noopMetricsReader) Read(_ context.Context) (map[string]float64, error) {
	return nil, nil
}
