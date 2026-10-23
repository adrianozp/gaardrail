package step

import (
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/pkg/control"
)

type Controller struct {
	step *control.Step
}

func New(cfg config.Config) (*Controller, error) {
	s, err := control.NewStep(cfg.PID.Max)
	if err != nil {
		return nil, err
	}
	return &Controller{step: s}, nil
}

func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	out, _, err := c.step.Compute(measured, measureTime)
	if err != nil {
		return 0, err
	}
	metrics.Gauge(map[string]float64{"step_output": out})
	return out, nil
}

func (c *Controller) GetParams() entities.ControllerParams {
	out := c.step.Output()
	return entities.ControllerParams{Max: &out}
}

func (c *Controller) SetParams(p entities.ControllerParams) error {
	if p.Max == nil {
		return nil
	}
	return c.step.SetOutput(*p.Max)
}

func (c *Controller) Reset() {
	c.step.Reset()
}

func (c *Controller) Type() string {
	return "step"
}
