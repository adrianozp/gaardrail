package exporter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/adrianozp/gaardrail/pkg/config"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func init() {
	model.NameValidationScheme = model.UTF8Validation
}

type Reader struct {
	endpoint string
	mappings map[string]string
	client   *http.Client
}

func New(cfg config.Config) *Reader {
	return &Reader{
		endpoint: cfg.MetricsPoller.Endpoint,
		mappings: cfg.MetricsPoller.Mappings,
		client:   &http.Client{},
	}
}

func (r *Reader) Read(ctx context.Context) (entities.Metrics, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint, nil)
	if err != nil {
		return entities.Metrics{}, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return entities.Metrics{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return entities.Metrics{}, fmt.Errorf("exporter: unexpected status %d", resp.StatusCode)
	}

	parser := expfmt.NewTextParser(model.NameValidationScheme)
	mfs, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil && len(mfs) == 0 {
		return entities.Metrics{}, fmt.Errorf("exporter: parse error: %w", err)
	}

	metric := entities.Metrics{
		MeasureTime: clock.Now(),
		Metrics:     make(map[string]float64),
	}
	for source, domain := range r.mappings {
		mf, ok := mfs[source]
		if !ok || len(mf.GetMetric()) == 0 {
			continue
		}
		metric.Metrics[domain] = sampleValue(mf.GetMetric()[0])
	}

	return metric, nil
}

func sampleValue(m *dto.Metric) float64 {
	switch {
	case m.Gauge != nil:
		return m.Gauge.GetValue()
	case m.Counter != nil:
		return m.Counter.GetValue()
	case m.Untyped != nil:
		return m.Untyped.GetValue()
	default:
		return 0
	}
}
