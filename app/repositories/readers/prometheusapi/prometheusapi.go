package prometheusapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/rs/zerolog/log"
)

type Reader struct {
	endpoint string
	mappings map[string]string
	http     *http.Client
}

func New(endpoint string, mappings map[string]string) *Reader {
	return &Reader{
		endpoint: endpoint,
		mappings: mappings,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

type apiResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value [2]json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (r *Reader) Read(ctx context.Context) (entities.Metrics, error) {
	metrics := make(map[string]float64)
	for expr, name := range r.mappings {
		val, err := r.queryValue(ctx, expr)
		if err != nil {
			return entities.Metrics{}, fmt.Errorf("prometheusapi: %s: %w", name, err)
		}
		metrics[name] = val
	}
	log.Debug().Msgf("prometheusapi: read metrics: %v", metrics)
	return entities.Metrics{
		MeasureTime: clock.Now(),
		Metrics:     metrics,
	}, nil
}

func (r *Reader) queryValue(ctx context.Context, expr string) (float64, error) {
	reqURL := r.endpoint + "?query=" + url.QueryEscape(expr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	if len(apiResp.Data.Result) == 0 {
		return 0, fmt.Errorf("no results for query %q", expr)
	}

	var valStr string
	if err := json.Unmarshal(apiResp.Data.Result[0].Value[1], &valStr); err != nil {
		return 0, err
	}

	return strconv.ParseFloat(valStr, 64)
}
