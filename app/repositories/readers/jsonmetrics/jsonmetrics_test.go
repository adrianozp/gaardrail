package jsonmetrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/readers/jsonmetrics"
	"github.com/adrianozp/gaardrail/pkg/clock"
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

	clock.WithTime(time.Date(1995, 11, 11, 0, 0, 0, 0, time.UTC))
	mappings := map[string]string{
		"cpu_usage":          "cpu",
		"active_connections": "connections",
	}
	reader := jsonmetrics.New(srv.URL, mappings)

	result, err := reader.Read(context.Background())
	require.NoError(t, err)

	expected := entities.Metrics{
		Metrics: map[string]float64{
			"cpu":         0.55,
			"connections": 7,
		},
		MeasureTime: time.Date(1995, 11, 11, 0, 0, 0, 0, time.UTC),
	}
	require.Equal(t, expected, result)
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

	now := time.Date(1995, 11, 11, 0, 0, 0, 0, time.UTC)
	clock.WithTime(now)
	result, err := reader.Read(context.Background())

	expected := entities.Metrics{
		MeasureTime: now,
		Metrics:     make(map[string]float64),
	}

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
