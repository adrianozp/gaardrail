package smith

import (
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/pkg/control"
	"github.com/adrianozp/gaardrail/pkg/control/filter"
	"github.com/adrianozp/gaardrail/pkg/ptr"
)

type Controller struct {
	mu    sync.Mutex
	smith *control.Smith
}

func New(cfg config.Config) (*Controller, error) {
	c, err := control.NewSmith(paramsFromConfig(cfg.Smith))
	if err != nil {
		return nil, err
	}
	return &Controller{smith: c}, nil
}

func paramsFromConfig(s config.Smith) control.SmithParams {
	return control.SmithParams{
		Kp:             s.Kp,
		Ki:             s.Ki,
		Setpoint:       s.Setpoint,
		SetpointFilter: filter.Spec{Kind: filter.Kind(s.SetpointFilterType), Size: s.SetpointFilterSize},
		Model:          control.Model{K: s.ModelK, Tau: s.ModelTau, Theta: s.ModelTheta},
		SampleSeconds:  s.SampleSeconds,
		Limits:         control.Limits{Min: s.Min, Max: s.Max, IClamp: s.IClamp},
	}
}

func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	out, terms, err := c.smith.Compute(measured, measureTime)
	if err != nil {
		return 0, err
	}
	metrics.Gauge(gauges(terms, c.smith.Params()))
	return out, nil
}

func gauges(t control.Terms, p control.SmithParams) map[string]float64 {
	return map[string]float64{
		"smith_setpoint":   t.Setpoint,
		"smith_error":      t.Error,
		"smith_feedback":   t.Feedback,
		"smith_prediction": t.Prediction,
		"smith_p_term":     t.P,
		"smith_i_term":     t.I,
		"smith_output":     t.Output,
		"smith_kp":         p.Kp,
		"smith_ki":         p.Ki,
		"smith_i_clamp":    p.IClamp,
		"smith_max":        p.Max,
	}
}

func (c *Controller) GetParams() entities.ControllerParams {
	p := c.smith.Params()
	kind := string(p.SetpointFilter.Kind)
	return entities.ControllerParams{
		Kp: &p.Kp, Ki: &p.Ki,
		Min: &p.Min, Max: &p.Max, IClamp: &p.IClamp,
		Setpoint:           &p.Setpoint,
		SetpointFilterType: &kind,
		SetpointFilterSize: &p.SetpointFilter.Size,
		ModelK:             &p.Model.K,
		ModelTau:           &p.Model.Tau,
		ModelTheta:         &p.Model.Theta,
		SampleSeconds:      &p.SampleSeconds,
	}
}

func (c *Controller) SetParams(u entities.ControllerParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.smith.Params()
	ptr.Assign(&p.Kp, u.Kp)
	ptr.Assign(&p.Ki, u.Ki)
	ptr.Assign(&p.Min, u.Min)
	ptr.Assign(&p.Max, u.Max)
	ptr.Assign(&p.IClamp, u.IClamp)
	ptr.Assign(&p.Setpoint, u.Setpoint)
	ptr.Assign((*string)(&p.SetpointFilter.Kind), u.SetpointFilterType)
	ptr.Assign(&p.SetpointFilter.Size, u.SetpointFilterSize)
	ptr.Assign(&p.Model.K, u.ModelK)
	ptr.Assign(&p.Model.Tau, u.ModelTau)
	ptr.Assign(&p.Model.Theta, u.ModelTheta)
	ptr.Assign(&p.SampleSeconds, u.SampleSeconds)
	return c.smith.SetParams(p)
}

func (c *Controller) Reset() {
	c.smith.Reset()
}

func (c *Controller) Type() string {
	return "smith"
}
