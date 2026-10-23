package step_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/step"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(v float64) *float64 { return &v }

func newStep(t *testing.T, max float64) *step.Controller {
	t.Helper()
	c, err := step.New(config.Config{PID: config.PID{Max: max}})
	require.NoError(t, err)
	return c
}

func TestStepOutputsMax(t *testing.T) {
	out, err := newStep(t, 5).Compute(42, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 5.0, out)
}

func TestStepGetParamsReflectsMax(t *testing.T) {
	p := newStep(t, 5).GetParams()
	require.NotNil(t, p.Max)
	assert.Equal(t, 5.0, *p.Max)
}

func TestStepSetParamsNilSafe(t *testing.T) {
	c := newStep(t, 5)
	require.NoError(t, c.SetParams(entities.ControllerParams{}))
	assert.Equal(t, 5.0, *c.GetParams().Max)
}

func TestStepSetParamsUpdatesMax(t *testing.T) {
	c := newStep(t, 5)
	require.NoError(t, c.SetParams(entities.ControllerParams{Max: ptr(9)}))
	out, err := c.Compute(0, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 9.0, out)
}

func TestStepSetParamsRejectsNaN(t *testing.T) {
	c := newStep(t, 5)
	require.Error(t, c.SetParams(entities.ControllerParams{Max: ptr(math.NaN())}))
	assert.Equal(t, 5.0, *c.GetParams().Max)
}
