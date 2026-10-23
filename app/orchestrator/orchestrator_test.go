package orchestrator_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/orchestrator"
	"github.com/adrianozp/gaardrail/app/orchestrator/mocks"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func defaultCfg() config.Config {
	return config.Config{Orchestrator: config.Orchestrator{Rate: 100, Burst: 10, Workers: 2}}
}

type countingConsumer struct {
	calls atomic.Int64
	err   error
}

func (c *countingConsumer) Consume(ctx context.Context) (string, error) {
	c.calls.Add(1)
	return "", c.err
}

func (c *countingConsumer) Size() (int64, error) { return 0, nil }

func stop(t *testing.T, o *orchestrator.Orchestrator) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, o.Stop(ctx))
}

func TestSetDrainRate_UpdatesLimiterLimit(t *testing.T) {
	o := orchestrator.NewOrchestrator(mocks.NewConsumer(t), defaultCfg())
	require.NoError(t, o.SetDrainRate(42.0))
	assert.Equal(t, rate.Limit(42.0), o.Limiter().Limit())
}

func TestNewOrchestrator_StartsWithDefaultRate(t *testing.T) {
	o := orchestrator.NewOrchestrator(mocks.NewConsumer(t), defaultCfg())
	assert.Equal(t, rate.Limit(100), o.Limiter().Limit())
}

func TestRun_RetriesConsumeAfterFailure(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Consume", mock.Anything).Return("", errors.New("kafka unavailable")).Maybe()
	consumer.On("Size").Return(int64(0), nil).Maybe()
	o := orchestrator.NewOrchestrator(consumer, defaultCfg())
	_ = o.SetDrainRate(1000)
	require.NoError(t, o.Start(context.Background()))
	time.Sleep(100 * time.Millisecond)
	stop(t, o)
	assert.Greater(t, len(consumer.Calls), 1, "loop should have retried Consume after error")
}

func TestStop_WaitsForWorkersToExit(t *testing.T) {
	consumer := &countingConsumer{}
	o := orchestrator.NewOrchestrator(consumer, defaultCfg())
	require.NoError(t, o.Start(context.Background()))
	time.Sleep(50 * time.Millisecond)
	stop(t, o)
	after := consumer.calls.Load()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, after, consumer.calls.Load(), "no Consume after Stop returns")
}

func TestParentContextCancelStopsWorkersWithoutPanic(t *testing.T) {
	o := orchestrator.NewOrchestrator(&countingConsumer{}, defaultCfg())
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, o.Start(ctx))
	cancel()
	stop(t, o)
}

func TestStop_InterruptsZeroRateWait(t *testing.T) {
	o := orchestrator.NewOrchestrator(&countingConsumer{}, defaultCfg())
	require.NoError(t, o.SetDrainRate(0))
	require.NoError(t, o.Start(context.Background()))
	time.Sleep(20 * time.Millisecond)
	begin := time.Now()
	stop(t, o)
	assert.Less(t, time.Since(begin), 500*time.Millisecond)
}

func TestStop_InterruptsLimiterDelay(t *testing.T) {
	cfg := defaultCfg()
	cfg.Orchestrator.Burst = 1
	o := orchestrator.NewOrchestrator(&countingConsumer{}, cfg)
	require.NoError(t, o.SetDrainRate(1.5))
	require.NoError(t, o.Start(context.Background()))
	time.Sleep(20 * time.Millisecond)
	begin := time.Now()
	stop(t, o)
	assert.Less(t, time.Since(begin), 300*time.Millisecond)
}

func TestStop_BeforeStartIsNoOp(t *testing.T) {
	stop(t, orchestrator.NewOrchestrator(&countingConsumer{}, defaultCfg()))
}
