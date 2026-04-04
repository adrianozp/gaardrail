package prometheusapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/readers/prometheusapi"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const successBody = `{
	"status": "success",
	"data": {
		"resultType": "vector",
		"result": [{"metric": {}, "value": [1234567890, "47.5"]}]
	}
}`

func TestPrometheusAPIReader_Read_ReturnsMappedValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "rate(process_cpu_seconds_total[15s])*100", r.URL.Query().Get("query"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(successBody))
	}))
	defer srv.Close()

	clock.WithTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	reader := prometheusapi.New(srv.URL, map[string]string{
		"rate(process_cpu_seconds_total[15s])*100": "cpu",
	})

	result, err := reader.Read(context.Background())
	require.NoError(t, err)

	expected := entities.Metrics{
		MeasureTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Metrics:     map[string]float64{"cpu": 47.5},
	}
	require.Equal(t, expected, result)
}

func TestPrometheusAPIReader_Read_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reader := prometheusapi.New(srv.URL, map[string]string{"rate(cpu[15s])": "cpu"})

	_, err := reader.Read(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestPrometheusAPIReader_Read_EmptyResult_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":{"result":[]}}`))
	}))
	defer srv.Close()

	reader := prometheusapi.New(srv.URL, map[string]string{"nonexistent": "cpu"})

	_, err := reader.Read(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no results")
}
