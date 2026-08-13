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
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/rs/zerolog/log"
)

type Reader struct {
	endpoint string
	mappings map[string]string
	http     *http.Client
}

func New(cfg config.Config) *Reader {
	return &Reader{
		endpoint: cfg.MetricsPoller.Endpoint,
		mappings: cfg.MetricsPoller.Mappings,
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
	var measureTime time.Time
	for expr, name := range r.mappings {
		val, t, err := r.queryValue(ctx, expr)
		if err != nil {
			return entities.Metrics{}, fmt.Errorf("prometheusapi: %s: %w", name, err)
		}
		metrics[name] = val
		measureTime = t
	}
	log.Debug().Msgf("prometheusapi: read metrics: %v", metrics)
	return entities.Metrics{
		MeasureTime: measureTime,
		Metrics:     metrics,
	}, nil
}

func (r *Reader) queryValue(ctx context.Context, expr string) (float64, time.Time, error) {
	reqURL := r.endpoint + "?query=" + url.QueryEscape(expr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, time.Time{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, time.Time{}, err
	}

	if len(apiResp.Data.Result) == 0 {
		return 0, time.Time{}, fmt.Errorf("no results for query %q", expr)
	}

	var tsFloat float64
	if err := json.Unmarshal(apiResp.Data.Result[0].Value[0], &tsFloat); err != nil {
		return 0, time.Time{}, err
	}
	sec := int64(tsFloat)
	ts := time.Unix(sec, int64((tsFloat-float64(sec))*1e9)).UTC()

	var valStr string
	if err := json.Unmarshal(apiResp.Data.Result[0].Value[1], &valStr); err != nil {
		return 0, time.Time{}, err
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, time.Time{}, err
	}
	return val, ts, nil
}
