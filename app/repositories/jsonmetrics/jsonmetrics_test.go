package jsonmetrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adrianozp/gaardrail/app/repositories/jsonmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONMetricsReader_Read_ExtractsMappedFields(t *testing.T) {
	body := `{"cpu_usage": 0.55, "active_connections": 7, "ignored": 999}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	mappings := map[string]string{
		"cpu_usage":          "cpu",
		"active_connections": "connections",
	}
	reader := jsonmetrics.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.InDelta(t, 0.55, result["cpu"], 0.001)
	assert.InDelta(t, 7.0, result["connections"], 0.001)
	_, hasIgnored := result["ignored"]
	assert.False(t, hasIgnored, "unmapped field should not appear in result")
}

func TestJSONMetricsReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	reader := jsonmetrics.New(srv.URL, map[string]string{"cpu_usage": "cpu"})

	_, err := reader.Read(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestJSONMetricsReader_Read_MissingFieldSkipped(t *testing.T) {
	body := `{"other_field": 1.0}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	reader := jsonmetrics.New(srv.URL, map[string]string{"cpu_usage": "cpu"})

	result, err := reader.Read(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}
