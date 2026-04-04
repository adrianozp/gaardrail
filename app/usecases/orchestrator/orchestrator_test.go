package orchestrator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/adrianozp/gaardrail/app/usecases/orchestrator/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestSetDrainRate_UpdatesLimiterLimit(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	o := orchestrator.NewOrchestrator(consumer)
	err := o.SetDrainRate(42.0)
	require.NoError(t, err)
	assert.Equal(t, rate.Limit(42.0), o.Limiter().Limit())
}

func TestNewOrchestrator_StartsWithDefaultRate(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	o := orchestrator.NewOrchestrator(consumer)
	assert.Equal(t, rate.Limit(1), o.Limiter().Limit())
}

func TestStart_ExitsWhenContextCancelled(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Consume").Return("", nil).Maybe()

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithCancel(context.Background())
	err := o.Start(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case <-o.Done():
		// goroutine exited cleanly
	case <-time.After(200 * time.Millisecond):
		t.Fatal("orchestrator goroutine did not exit after context cancellation")
	}
}
func TestRun_LogsErrorOnConsumeFailure(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Consume").Return("", errors.New("kafka unavailable")).Maybe()

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := o.Start(ctx)
	require.NoError(t, err)
	<-ctx.Done()

	callCount := 0
	for _, c := range consumer.Calls {
		if c.Method == "Consume" {
			callCount++
		}
	}
	assert.Greater(t, callCount, 1, "loop should have retried Consume after error")
}
