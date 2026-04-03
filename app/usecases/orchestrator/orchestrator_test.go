package orchestrator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/adrianozp/gaardrail/app/usecases/orchestrator/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestNewOrchestrator_StartsWithZeroRate(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	o := orchestrator.NewOrchestrator(consumer)
	assert.Equal(t, rate.Limit(0), o.Limiter().Limit())
}

func TestStart_ExitsWhenContextCancelled(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Size").Return(int64(0), nil).Maybe()

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

func TestRun_ConsumesWhenQueueNonEmpty(t *testing.T) {
	consumed := make(chan struct{}, 1)
	consumer := mocks.NewConsumer(t)
	consumer.On("Size").Return(int64(1), nil).Maybe()
	consumer.On("Consume").Return("msg-1", nil).
		Run(func(_ mock.Arguments) {
			select {
			case consumed <- struct{}{}:
			default:
			}
		}).Maybe()

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := o.Start(ctx)
	require.NoError(t, err)

	select {
	case <-consumed:
		// success
	case <-ctx.Done():
		t.Fatal("expected Consume to be called but it was not")
	}
}

func TestRun_SkipsConsumeWhenQueueEmpty(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Size").Return(int64(0), nil)

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := o.Start(ctx)
	require.NoError(t, err)

	<-ctx.Done()
	consumer.AssertNotCalled(t, "Consume")
}

func TestRun_LogsErrorOnConsumeFailure(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Size").Return(int64(1), nil).Maybe()
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

func TestRun_LogsErrorOnSizeFailure(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Size").Return(int64(0), errors.New("kafka unavailable")).Maybe()

	o := orchestrator.NewOrchestrator(consumer)
	_ = o.SetDrainRate(1000)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := o.Start(ctx)
	require.NoError(t, err)
	<-ctx.Done()

	sizeCallCount := 0
	for _, c := range consumer.Calls {
		if c.Method == "Size" {
			sizeCallCount++
		}
	}
	assert.Greater(t, sizeCallCount, 1, "loop should have retried after Size error")
	consumer.AssertNotCalled(t, "Consume")
}
