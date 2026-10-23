package pid

import (
	"math"
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
	mu          sync.Mutex
	pid         *control.PID
	feedforward bool
}

func New(cfg config.Config, feedforward bool) (*Controller, error) {
	p := paramsFromConfig(cfg)
	if !feedforward {
		p.FFGain = 0
	}
	c, err := control.NewPID(p)
	if err != nil {
		return nil, err
	}
	return &Controller{pid: c, feedforward: feedforward}, nil
}

func paramsFromConfig(cfg config.Config) control.PIDParams {
	p := cfg.PID
	return control.PIDParams{
		Kp:             p.Kp,
		Ki:             p.Ki,
		Kd:             p.Kd,
		FFGain:         p.FfGain,
		Setpoint:       p.Setpoint,
		SetpointFilter: setpointFilter(p, cfg.MetricsPoller.IntervalMs),
		Limits:         control.Limits{Min: p.Min, Max: p.Max, IClamp: p.IClamp},
	}
}

func setpointFilter(p config.PID, intervalMs int) filter.Spec {
	if p.SetpointFilterType == "" && p.SetpointFilterTau > 0 {
		return filter.Spec{Kind: filter.Exponential, Size: equivalentSamples(p.SetpointFilterTau, intervalMs)}
	}
	return filter.Spec{Kind: filter.Kind(p.SetpointFilterType), Size: p.SetpointFilterSize}
}

func equivalentSamples(tauSeconds float64, intervalMs int) int {
	if intervalMs <= 0 {
		return 1
	}
	return max(int(math.Round(tauSeconds*1000/float64(intervalMs))), 1)
}

func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	out, terms, err := c.pid.Compute(measured, measureTime)
	if err != nil {
		return 0, err
	}
	metrics.Gauge(gauges(terms, c.pid.Params()))
	return out, nil
}

func gauges(t control.Terms, p control.PIDParams) map[string]float64 {
	return map[string]float64{
		"pid_setpoint": t.Setpoint,
		"pid_measured": t.Measured,
		"pid_error":    t.Error,
		"pid_p_term":   t.P,
		"pid_i_term":   t.I,
		"pid_d_term":   t.D,
		"pid_ff_term":  t.FF,
		"pid_output":   t.FF + t.P + t.I + t.D,
		"pid_kp":       p.Kp,
		"pid_ki":       p.Ki,
		"pid_kd":       p.Kd,
		"pid_i_clamp":  p.IClamp,
		"pid_max":      p.Max,
	}
}

func (c *Controller) GetParams() entities.ControllerParams {
	p := c.pid.Params()
	kind := string(p.SetpointFilter.Kind)
	params := entities.ControllerParams{
		Kp: &p.Kp, Ki: &p.Ki, Kd: &p.Kd,
		Min: &p.Min, Max: &p.Max, IClamp: &p.IClamp,
		Setpoint:           &p.Setpoint,
		SetpointFilterType: &kind,
		SetpointFilterSize: &p.SetpointFilter.Size,
	}
	if c.feedforward {
		params.FfGain = &p.FFGain
	}
	return params
}

func (c *Controller) SetParams(u entities.ControllerParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.pid.Params()
	ptr.Assign(&p.Kp, u.Kp)
	ptr.Assign(&p.Ki, u.Ki)
	ptr.Assign(&p.Kd, u.Kd)
	ptr.Assign(&p.Min, u.Min)
	ptr.Assign(&p.Max, u.Max)
	ptr.Assign(&p.IClamp, u.IClamp)
	ptr.Assign(&p.Setpoint, u.Setpoint)
	ptr.Assign((*string)(&p.SetpointFilter.Kind), u.SetpointFilterType)
	ptr.Assign(&p.SetpointFilter.Size, u.SetpointFilterSize)
	if c.feedforward {
		ptr.Assign(&p.FFGain, u.FfGain)
	}
	return c.pid.SetParams(p)
}

func (c *Controller) Reset() {
	c.pid.Reset()
}

func (c *Controller) Type() string {
	if c.feedforward {
		return "pidff"
	}
	return "pid"
}
