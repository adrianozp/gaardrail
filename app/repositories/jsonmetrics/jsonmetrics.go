package jsonmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// JSONMetricsReader scrapes a flat JSON object endpoint and maps source field
// names to domain names via the configured Mappings.
// All JSON values must be numeric (float64). Non-numeric fields are ignored.
type JSONMetricsReader struct {
	endpoint string
	mappings map[string]string
	client   *http.Client
}

func New(endpoint string, mappings map[string]string) *JSONMetricsReader {
	return &JSONMetricsReader{
		endpoint: endpoint,
		mappings: mappings,
		client:   &http.Client{},
	}
}

func (r *JSONMetricsReader) Read(ctx context.Context) (map[string]float64, error) {
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
		return nil, fmt.Errorf("jsonmetrics: unexpected status %d", resp.StatusCode)
	}

	var raw map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("jsonmetrics: decode error: %w", err)
	}

	result := make(map[string]float64)
	for source, domain := range r.mappings {
		if v, ok := raw[source]; ok {
			result[domain] = v
		}
	}

	return result, nil
}
