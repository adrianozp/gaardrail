package smith_test

import (
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/smith"
	"github.com/adrianozp/gaardrail/internal/metrics/metricstest"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func smithConfig() config.Config {
	return config.Config{Smith: config.Smith{Kp: 1, Ki: 0.1, Min: 0, Max: 100, IClamp: 50, Setpoint: 50,
		ModelK: 2, ModelTau: 10, ModelTheta: 10, SampleSeconds: 10}}
}

func newController(t *testing.T) *smith.Controller {
	t.Helper()
	c, err := smith.New(smithConfig())
	require.NoError(t, err)
	return c
}

func TestNew_InvalidModelReturnsError(t *testing.T) {
	cfg := smithConfig()
	cfg.Smith.SampleSeconds = 0
	_, err := smith.New(cfg)
	require.Error(t, err)
}

func TestGetParams_IncludesModel(t *testing.T) {
	p := newController(t).GetParams()
	assert.Equal(t, 2.0, *p.ModelK)
	assert.Equal(t, 10.0, *p.ModelTau)
	assert.Equal(t, 10.0, *p.ModelTheta)
	assert.Equal(t, 10.0, *p.SampleSeconds)
	assert.Nil(t, p.Kd)
	assert.Nil(t, p.FfGain)
}

func TestSetParams_PartialModelUpdate(t *testing.T) {
	c := newController(t)
	require.NoError(t, c.SetParams(entities.ControllerParams{ModelTheta: ptr(20.0), Kp: ptr(2.0)}))
	p := c.GetParams()
	assert.Equal(t, 20.0, *p.ModelTheta)
	assert.Equal(t, 2.0, *p.ModelK)
	assert.Equal(t, 2.0, *p.Kp)
}

func TestSetParams_InvalidModelIsRejected(t *testing.T) {
	c := newController(t)
	require.Error(t, c.SetParams(entities.ControllerParams{ModelTau: ptr(-1.0)}))
	assert.Equal(t, 10.0, *c.GetParams().ModelTau)
}

func TestCompute_PublishesTermsAsGauges(t *testing.T) {
	rec := metricstest.Capture(t)
	c := newController(t)
	out, err := c.Compute(40, time.Now())
	require.NoError(t, err)
	got, ok := rec.Value("smith_output")
	assert.True(t, ok)
	assert.Equal(t, out, got)
	got, _ = rec.Value("smith_error")
	assert.Equal(t, 10.0, got)
}
