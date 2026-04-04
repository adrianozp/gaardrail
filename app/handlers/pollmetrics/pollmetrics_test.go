package pollmetrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics"
	"github.com/adrianozp/gaardrail/app/handlers/pollmetrics/mocks"
	repomocks "github.com/adrianozp/gaardrail/app/repositories/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPollingHandler_CallsReadAndProcessOnEachTick(t *testing.T) {
	mockReader := repomocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	ticked := make(chan struct{}, 3)

	mockReader.On("Read", mock.Anything).
		Return(map[string]float64{"cpu": 0.4}, nil)
	mockProcess.On("Process", mock.MatchedBy(func(m entities.Metrics) bool {
		return m.Metrics["cpu"] == 0.4
	})).Run(func(_ mock.Arguments) {
		ticked <- struct{}{}
	}).Return(nil)

	ctx := t.Context()

	handler := pollmetrics.New(mockReader, mockProcess, 5*time.Millisecond)
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
	mockReader := repomocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	processed := make(chan struct{}, 1)

	mockReader.On("Read", mock.Anything).
		Return(map[string]float64(nil), errors.New("read failed")).Once()
	mockReader.On("Read", mock.Anything).
		Return(map[string]float64{"cpu": 0.5}, nil)
	mockProcess.On("Process", mock.MatchedBy(func(m entities.Metrics) bool {
		return m.Metrics["cpu"] == 0.5
	})).Run(func(_ mock.Arguments) {
		processed <- struct{}{}
	}).Return(nil)

	ctx := t.Context()

	handler := pollmetrics.New(mockReader, mockProcess, 5*time.Millisecond)
	require.NoError(t, handler.Start(ctx))

	select {
	case <-processed:
		// success: loop continued past the error
	case <-time.After(500 * time.Millisecond):
		t.Fatal("process was never called after read error — loop may have stopped")
	}
}

func TestPollingHandler_ContextCancellationExitsCleanly(t *testing.T) {
	mockReader := repomocks.NewMetricsReader(t)
	mockProcess := mocks.NewProcessMetrics(t)

	ctx, cancel := context.WithCancel(context.Background())

	// long interval — we cancel before the first tick
	handler := pollmetrics.New(mockReader, mockProcess, 1*time.Hour)
	require.NoError(t, handler.Start(ctx))

	cancel()

	select {
	case <-handler.Done():
		// success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("goroutine did not exit after context cancellation")
	}
}
