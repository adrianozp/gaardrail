package pid_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/pid"
	"github.com/adrianozp/gaardrail/internal/metrics/metricstest"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var base = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

func pidConfig() config.Config {
	return config.Config{PID: config.PID{Kp: 1, Ki: 0.5, Min: 0, Max: 100, IClamp: 50, Setpoint: 50, FfGain: 2.5}}
}

func newController(t *testing.T, cfg config.Config, feedforward bool) *pid.Controller {
	t.Helper()
	c, err := pid.New(cfg, feedforward)
	require.NoError(t, err)
	return c
}

func TestNew_InvalidLimitsReturnError(t *testing.T) {
	cfg := pidConfig()
	cfg.PID.Min = 200
	_, err := pid.New(cfg, false)
	require.Error(t, err)
}

func TestNew_UnknownSetpointFilterReturnsError(t *testing.T) {
	cfg := pidConfig()
	cfg.PID.SetpointFilterType = "median"
	_, err := pid.New(cfg, true)
	require.Error(t, err)
}

func TestFeedforwardOnlyForPIDFF(t *testing.T) {
	plain := newController(t, pidConfig(), false)
	withFF := newController(t, pidConfig(), true)
	outPlain, err := plain.Compute(50, base)
	require.NoError(t, err)
	outFF, err := withFF.Compute(50, base)
	require.NoError(t, err)
	assert.InDelta(t, 0, outPlain, 1e-9)
	assert.InDelta(t, 20, outFF, 1e-9)
	assert.Equal(t, "pid", plain.Type())
	assert.Equal(t, "pidff", withFF.Type())
}

func TestPlainPIDIgnoresFeedforwardGain(t *testing.T) {
	c := newController(t, pidConfig(), false)
	require.NoError(t, c.SetParams(entities.ControllerParams{FfGain: ptr(1.0)}))
	assert.Nil(t, c.GetParams().FfGain)
	out, err := c.Compute(50, base)
	require.NoError(t, err)
	assert.InDelta(t, 0, out, 1e-9)
}

func TestSetParams_PartialUpdateKeepsOtherFields(t *testing.T) {
	c := newController(t, pidConfig(), true)
	require.NoError(t, c.SetParams(entities.ControllerParams{Kp: ptr(3.0), SetpointFilterType: ptr("moving_average")}))
	p := c.GetParams()
	assert.Equal(t, 3.0, *p.Kp)
	assert.Equal(t, 0.5, *p.Ki)
	assert.Equal(t, 50.0, *p.Setpoint)
	assert.Equal(t, 2.5, *p.FfGain)
	assert.Equal(t, "moving_average", *p.SetpointFilterType)
	assert.Equal(t, 1, *p.SetpointFilterSize)
}

func TestSetParams_InvalidUpdateIsRejected(t *testing.T) {
	c := newController(t, pidConfig(), true)
	require.Error(t, c.SetParams(entities.ControllerParams{Min: ptr(500.0)}))
	assert.Equal(t, 0.0, *c.GetParams().Min)
}

func TestLegacyTauMapsToEquivalentExponential(t *testing.T) {
	cfg := config.Config{MetricsPoller: config.MetricsPoller{IntervalMs: 5000},
		PID: config.PID{Kp: 1, Min: -100, Max: 100, IClamp: 100, Setpoint: 50, SetpointFilterTau: 10}}
	c := newController(t, cfg, false)
	out, err := c.Compute(0, base)
	require.NoError(t, err)
	assert.InDelta(t, (1-math.Exp(-0.5))*50, out, 1e-9)
	assert.Equal(t, "exponential", *c.GetParams().SetpointFilterType)
	assert.Equal(t, 2, *c.GetParams().SetpointFilterSize)
}

func TestCompute_PublishesTermsAsGauges(t *testing.T) {
	rec := metricstest.Capture(t)
	c := newController(t, pidConfig(), true)
	_, err := c.Compute(40, base)
	require.NoError(t, err)
	_, err = c.Compute(40, base.Add(2*time.Second))
	require.NoError(t, err)
	want := map[string]float64{"pid_setpoint": 50, "pid_measured": 40, "pid_error": 10,
		"pid_p_term": 10, "pid_i_term": 10, "pid_ff_term": 20, "pid_output": 40, "pid_kp": 1}
	for name, v := range want {
		got, ok := rec.Value(name)
		assert.True(t, ok, name)
		assert.InDelta(t, v, got, 1e-9, name)
	}
}
