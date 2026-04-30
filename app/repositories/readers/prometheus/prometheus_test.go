package prometheus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/readers/prometheus"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCfg(endpoint string, mappings map[string]string) config.Config {
	return config.Config{
		MetricsPoller: config.MetricsPoller{
			Endpoint: endpoint,
			Mappings: mappings,
		},
	}
}

const prometheusBody = `# HELP process_cpu_seconds_total Total CPU time.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.42
# HELP mysql_global_status_threads_connected Open connections.
# TYPE mysql_global_status_threads_connected gauge
mysql_global_status_threads_connected 12
# HELP ignored_metric Not in mapping.
# TYPE ignored_metric gauge
ignored_metric 99
`

func TestPrometheusMetricsReader_Read_ExtractsMappedMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(prometheusBody))
	}))
	defer srv.Close()

	clock.WithTime(time.Date(1995, 11, 11, 0, 0, 0, 0, time.UTC))

	mappings := map[string]string{
		"process_cpu_seconds_total":             "cpu",
		"mysql_global_status_threads_connected": "connections",
	}
	reader := prometheus.New(newCfg(srv.URL, mappings))

	result, err := reader.Read(context.Background())
	require.NoError(t, err)

	expected := entities.Metrics{
		Metrics: map[string]float64{
			"cpu":         0.42,
			"connections": 12,
		},
		MeasureTime: time.Date(1995, 11, 11, 0, 0, 0, 0, time.UTC),
	}
	require.Equal(t, expected, result)
}

func TestPrometheusMetricsReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reader := prometheus.New(newCfg(srv.URL, map[string]string{"cpu_total": "cpu"}))

	_, err := reader.Read(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestPrometheusMetricsReader_Read_MissingMetricSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(prometheusBody))
	}))
	defer srv.Close()

	now := time.Date(1995, 11, 11, 0, 0, 0, 0, time.UTC)
	clock.WithTime(now)
	mappings := map[string]string{
		"nonexistent_metric": "cpu",
	}
	reader := prometheus.New(newCfg(srv.URL, mappings))

	result, err := reader.Read(context.Background())

	expected := entities.Metrics{
		MeasureTime: now,
		Metrics:     make(map[string]float64),
	}
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
