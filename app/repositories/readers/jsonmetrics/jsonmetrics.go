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
		return entities.Metrics{}, fmt.Errorf("jsonmetrics: unexpected status %d", resp.StatusCode)
	}

	var raw map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return entities.Metrics{}, fmt.Errorf("jsonmetrics: decode error: %w", err)
	}

	metric := entities.Metrics{
		MeasureTime: clock.Now(),
		Metrics:     make(map[string]float64),
	}
	for source, domain := range r.mappings {
		if v, ok := raw[source]; ok {
			metric.Metrics[domain] = v
		}
	}

	return metric, nil
}
