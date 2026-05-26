package step

import (
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// Step always outputs Max, ignoring the measured value.
type Step struct {
	mu  sync.RWMutex
	max float64
}

func NewStep(cfg config.Config) *Step {
	return &Step{max: cfg.PID.Max}
}

func (c *Step) Compute(_ float64, _ time.Time) (float64, error) {
	c.mu.RLock()
	max := c.max
	c.mu.RUnlock()

	metrics.Gauge(map[string]float64{"step_output": max})
	return max, nil
}

func (c *Step) GetParams() entities.ControllerParams {
	c.mu.RLock()
	defer c.mu.RUnlock()

	max := c.max
	return entities.ControllerParams{Max: &max}
}

// SetParams updates the step output. Only non-nil fields are applied, so PATCH
// bodies that omit unchanged fields are safe.
func (c *Step) SetParams(p entities.ControllerParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p.Max != nil {
		c.max = *p.Max
	}
	return nil
}

func (c *Step) SetSetpoint(setpoint float64) error {
	return nil
}

func (c *Step) Reset() {
}

func (c *Step) Type() string { return "step" }
