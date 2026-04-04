package prometheus

import (
	"context"
	"fmt"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// PrometheusMetricsReader scrapes a Prometheus text-format endpoint and maps
// source metric names to domain names via the configured Mappings.
// Only the first sample of each matched metric family is used (suitable for
// no-label or single-instance metrics such as those from mysqld_exporter or
// node_exporter).
type PrometheusMetricsReader struct {
	endpoint string
	mappings map[string]string
	client   *http.Client
}

func New(endpoint string, mappings map[string]string) *PrometheusMetricsReader {
	return &PrometheusMetricsReader{
		endpoint: endpoint,
		mappings: mappings,
		client:   &http.Client{},
	}
}

func (r *PrometheusMetricsReader) Read(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus: unexpected status %d", resp.StatusCode)
	}

	var parser expfmt.TextParser
	mfs, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil && len(mfs) == 0 {
		return nil, fmt.Errorf("prometheus: parse error: %w", err)
	}

	result := make(map[string]float64)
	for source, domain := range r.mappings {
		mf, ok := mfs[source]
		if !ok || len(mf.GetMetric()) == 0 {
			continue
		}
		result[domain] = sampleValue(mf.GetMetric()[0])
	}

	return result, nil
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
