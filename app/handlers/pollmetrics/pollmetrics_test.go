package pollmetrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics/mocks"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPollingHandler_CallsReadAndProcessOnEachTick(t *testing.T) {
	mockReader := mocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	ticked := make(chan struct{}, 3)

	metric := entities.Metrics{
		Metrics: map[string]float64{"cpu": 0.4},
	}
	mockReader.On("Read", mock.Anything).Return(metric, nil)
	mockProcess.On("Process", metric).Run(func(_ mock.Arguments) {
		ticked <- struct{}{}
	}).Return(nil)

	ctx := t.Context()

	handler := pollmetrics.New(mockReader, mockProcess, config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 5}})
	require.NoError(t, handler.Start(ctx))

	for i := range 2 {
		select {
		case <-ticked:
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("tick %d never happened", i+1)
		}
	}
}

func TestPollingHandler_ReadErrorDoesNotStopLoop(t *testing.T) {
	mockReader := mocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	processed := make(chan struct{}, 1)

	emptyMetric := entities.Metrics{
		Metrics: map[string]float64(nil),
	}
	metric := entities.Metrics{
		Metrics: map[string]float64{"cpu": 0.5},
	}

	mockReader.On("Read", mock.Anything).Return(emptyMetric, errors.New("read failed")).Once()
	mockReader.On("Read", mock.Anything).Return(metric, nil)
	mockProcess.On("Process", metric).Run(func(_ mock.Arguments) {
		processed <- struct{}{}
	}).Return(nil)

	ctx := t.Context()

	handler := pollmetrics.New(mockReader, mockProcess, config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 5}})
	require.NoError(t, handler.Start(ctx))

	select {
	case <-processed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("process was never called after read error — loop may have stopped")
	}
}

func TestPollingHandler_ContextCancellationExitsCleanly(t *testing.T) {
	mockReader := mocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	ctx, cancel := context.WithCancel(context.Background())

	handler := pollmetrics.New(mockReader, mockProcess, config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 3600000}})
	require.NoError(t, handler.Start(ctx))

	cancel()

	select {
	case <-handler.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("goroutine did not exit after context cancellation")
	}
}

func TestPollingHandler_StopExitsLoop(t *testing.T) {
	handler := pollmetrics.New(mocks.NewMetricsReader(t), mocks.NewProcessMetrics(t),
		config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 60000}})
	require.NoError(t, handler.Start(context.Background()))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, handler.Stop(ctx))
	select {
	case <-handler.Done():
	default:
		t.Fatal("loop still running after Stop returned")
	}
}

func TestPollingHandler_StopBeforeStartIsNoOp(t *testing.T) {
	handler := pollmetrics.New(mocks.NewMetricsReader(t), mocks.NewProcessMetrics(t), config.Config{})
	require.NoError(t, handler.Stop(context.Background()))
}
