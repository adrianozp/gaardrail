package prometheus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adrianozp/gaardrail/app/repositories/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	mappings := map[string]string{
		"process_cpu_seconds_total":             "cpu",
		"mysql_global_status_threads_connected": "connections",
	}
	reader := prometheus.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.InDelta(t, 0.42, result["cpu"], 0.001)
	assert.InDelta(t, 12.0, result["connections"], 0.001)
	_, hasIgnored := result["ignored_metric"]
	assert.False(t, hasIgnored, "unmapped metric should not appear in result")
}

func TestPrometheusMetricsReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reader := prometheus.New(srv.URL, map[string]string{"cpu_total": "cpu"})

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

	mappings := map[string]string{
		"nonexistent_metric": "cpu",
	}
	reader := prometheus.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}
