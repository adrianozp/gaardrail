package readers

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/entities"
	cloudwatchrepo "github.com/adrianozp/gaardrail/app/repositories/readers/cloudwatch"
	jsonmetricsrepo "github.com/adrianozp/gaardrail/app/repositories/readers/jsonmetrics"
	prometheusrepo "github.com/adrianozp/gaardrail/app/repositories/readers/prometheus"
	prometheusapirepo "github.com/adrianozp/gaardrail/app/repositories/readers/prometheusapi"
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
	case "prometheusapi":
		return prometheusapirepo.New(cfg.MetricsPoller.Endpoint, cfg.MetricsPoller.Mappings), nil
	case "cloudwatch":
		return cloudwatchrepo.New(cfg.CloudWatch, cfg.MetricsPoller.Mappings)
	default:
		return nil, fmt.Errorf("metricspoller: unknown protocol %q", cfg.MetricsPoller.Protocol)
	}
}

// noopMetricsReader is used when MetricsPoller.Enabled is false.
type noopMetricsReader struct{}

func (n *noopMetricsReader) Read(_ context.Context) (entities.Metrics, error) {
	return entities.Metrics{}, nil
}
