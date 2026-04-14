package orchestrator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/orchestrator"
	"github.com/adrianozp/gaardrail/app/orchestrator/mocks"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)


func defaultCfg() config.Config {
	return config.Config{Orchestrator: config.Orchestrator{Rate: 100, Burst: 10, Workers: 1}}
}

func TestSetDrainRate_UpdatesLimiterLimit(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	o := orchestrator.NewOrchestrator(consumer, defaultCfg())
	err := o.SetDrainRate(42.0)
	require.NoError(t, err)
	assert.Equal(t, rate.Limit(42.0), o.Limiter().Limit())
}

func TestNewOrchestrator_StartsWithDefaultRate(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	o := orchestrator.NewOrchestrator(consumer, defaultCfg())
	assert.Equal(t, rate.Limit(100), o.Limiter().Limit())
}

func TestRun_LogsErrorOnConsumeFailure(t *testing.T) {
	consumer := mocks.NewConsumer(t)
	consumer.On("Consume").Return("", errors.New("kafka unavailable")).Maybe()
	consumer.On("Size").Return(int64(0), nil).Maybe()

	o := orchestrator.NewOrchestrator(consumer, defaultCfg())
	_ = o.SetDrainRate(1000)

	err := o.Start(context.Background())
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	callCount := 0
	for _, c := range consumer.Calls {
		if c.Method == "Consume" {
			callCount++
		}
	}
	assert.Greater(t, callCount, 1, "loop should have retried Consume after error")
}
