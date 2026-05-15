package step

import (
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// StepController always outputs Max, ignoring the measured value.
type Step struct {
	max float64
}

func NewStep(cfg config.Config) *Step {
	return &Step{max: cfg.PID.Max}
}

func (c *Step) Compute(measured float64, _ time.Time) (float64, error) {
	metrics.Gauge(map[string]float64{
		"step_measured": measured,
		"step_output":   c.max,
	})
	return c.max, nil
}

func (c *Step) GetParams() entities.ControllerParams {
	return entities.ControllerParams{}
}

func (c *Step) SetParams(p entities.ControllerParams) error {
	c.max = *p.Max
	return nil
}

func (c *Step) SetSetpoint(setpoint float64) error {
	return nil
}

func (c *Step) Reset() {
}
