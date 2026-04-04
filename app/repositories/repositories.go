package repositories

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/entities"
	jsonmetricsrepo "github.com/adrianozp/gaardrail/app/repositories/jsonmetrics"
	prometheusrepo "github.com/adrianozp/gaardrail/app/repositories/prometheus"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type MetricsReader interface {
	Read(ctx context.Context) (entities.Metrics, error)
}

func NewMetricsReader(cfg config.Config) (MetricsReader, error) {
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

func (n *noopMetricsReader) Read(_ context.Context) (entities.Metrics, error) {
	return entities.Metrics{}, nil
}
