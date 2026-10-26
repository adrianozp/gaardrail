package switchable

import (
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeController struct {
	name       string
	resetCalls int
	output     float64
}

func (f *fakeController) Compute(float64, time.Time) (float64, error) { return f.output, nil }
func (f *fakeController) GetParams() entities.ControllerParams        { return entities.ControllerParams{} }
func (f *fakeController) SetParams(entities.ControllerParams) error   { return nil }
func (f *fakeController) Reset()                                      { f.resetCalls++ }
func (f *fakeController) Type() string                                { return f.name }

func newWith(active string, a, b *fakeController) *Controller {
	return &Controller{
		active:     map[string]controllers.Controller{"pid": a, "step": b}[active],
		activeType: active,
		available:  map[string]controllers.Controller{"pid": a, "step": b},
	}
}

func TestSetType_SwitchesActiveAndResets(t *testing.T) {
	pid := &fakeController{name: "pid", output: 1}
	step := &fakeController{name: "step", output: 2}
	c := newWith("pid", pid, step)

	require.NoError(t, c.SetType("step"))

	assert.Equal(t, "step", c.Type())
	assert.Equal(t, 1, step.resetCalls, "newly activated controller must be reset")
	assert.Equal(t, 0, pid.resetCalls, "previous controller must not be reset")

	out, err := c.Compute(0, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 2.0, out, "Compute must delegate to the newly active controller")
}

func TestSetType_SameTypeIsNoOp(t *testing.T) {
	pid := &fakeController{name: "pid"}
	step := &fakeController{name: "step"}
	c := newWith("pid", pid, step)

	require.NoError(t, c.SetType("pid"))

	assert.Equal(t, "pid", c.Type())
	assert.Equal(t, 0, pid.resetCalls, "no-op switch must not reset")
}

func TestSetType_UnknownTypeErrorsAndKeepsActive(t *testing.T) {
	pid := &fakeController{name: "pid"}
	step := &fakeController{name: "step"}
	c := newWith("pid", pid, step)

	err := c.SetType("banana")

	require.Error(t, err)
	assert.Equal(t, "pid", c.Type(), "active controller must be unchanged on error")
}

func defaultConfig(t *testing.T, controllerType string) config.Config {
	t.Helper()
	cfg := config.Config{}
	require.NoError(t, defaults.Set(&cfg))
	cfg.Controller.Type = controllerType
	return cfg
}

func TestNew_SelectsConfiguredController(t *testing.T) {
	for _, name := range []string{"pid", "pidff", "step", "smith"} {
		c, err := New(defaultConfig(t, name))
		require.NoError(t, err)
		assert.Equal(t, name, c.Type())
		assert.Equal(t, name, c.current().Type())
	}
}

func TestNew_UnknownTypeErrors(t *testing.T) {
	_, err := New(defaultConfig(t, "banana"))
	require.Error(t, err)
}

func TestNew_InvalidControllerConfigErrors(t *testing.T) {
	cfg := defaultConfig(t, "pid")
	cfg.Smith.ModelTau = 0
	_, err := New(cfg)
	require.ErrorContains(t, err, "controller smith")
}
