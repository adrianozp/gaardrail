package jsonmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type JSONMetricsReader struct {
	endpoint string
	mappings map[string]string
	client   *http.Client
}

func New(cfg config.Config) *JSONMetricsReader {
	return &JSONMetricsReader{
		endpoint: cfg.MetricsPoller.Endpoint,
		mappings: cfg.MetricsPoller.Mappings,
		client:   &http.Client{},
	}
}

func (r *JSONMetricsReader) Read(ctx context.Context) (entities.Metrics, error) {
	raw, err := r.fetch(ctx)
	if err != nil {
		return entities.Metrics{}, err
	}
	metric := entities.Metrics{MeasureTime: clock.Now(), Metrics: make(map[string]float64)}
	for source, domain := range r.mappings {
		if v, ok := raw[source]; ok {
			metric.Metrics[domain] = v
		}
	}
	return metric, nil
}

func (r *JSONMetricsReader) fetch(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jsonmetrics: unexpected status %d", resp.StatusCode)
	}
	var raw map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("jsonmetrics: decode error: %w", err)
	}
	return raw, nil
}
