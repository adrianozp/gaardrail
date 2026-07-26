package pid

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/internal/filter"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// Controller is a discrete PID controller in position form.
// output(t) = clamp(P + I(t) + D, Min, Max)
// This is the absolute output (e.g. drain rate RPS), not a delta.
type Controller struct {
	mu         sync.RWMutex
	Kp, Ki, Kd float64
	Min, Max   float64 // output clamp
	// IClamp is the anti-windup bound for the integral term.
	// Should be <= Max for correct saturation behavior.
	IClamp float64
	// Kff is the plant static gain used for feedforward (u_ff = setpoint/Kff).
	// Zero disables feedforward; the feedforward sets the operating point so the
	// integral does not have to ramp up to it (removes setpoint overshoot).
	Kff float64

	setpoint     float64
	i            float64
	prevE        float64
	first        bool
	lastCompute  time.Time
	refFilter    *filter.Signal
	refEffective float64
}

type ControllerParams struct {
	Kp       float64
	Ki       float64
	Kd       float64
	Min      float64
	Max      float64
	IClamp   float64
	Setpoint float64
}

// New creates a Controller. iClamp prevents integral windup.
func New(cfg config.Config) *Controller {
	if cfg.PID.Min > cfg.PID.Max {
		panic("pid.New: min must be <= max")
	}
	if cfg.PID.IClamp < 0 {
		panic("pid.New: iClamp must be >= 0")
	}
	return &Controller{
		Kp:        cfg.PID.Kp,
		Ki:        cfg.PID.Ki,
		Kd:        cfg.PID.Kd,
		Min:       cfg.PID.Min,
		Max:       cfg.PID.Max,
		IClamp:    cfg.PID.IClamp,
		Kff:       cfg.PID.FfGain,
		first:     true,
		setpoint:  cfg.PID.Setpoint,
		refFilter: referenceFilterFromConfig(cfg),
	}
}

func referenceFilterFromConfig(cfg config.Config) *filter.Signal {
	p := cfg.PID
	kind, size := p.SetpointFilterType, p.SetpointFilterSize
	if kind == "" && p.SetpointFilterTau > 0 {
		kind = "exponential"
		size = equivalentSamples(p.SetpointFilterTau, cfg.MetricsPoller.IntervalMs)
	}
	f, err := filter.NewReferenceFilter(kind, size)
	if err != nil {
		panic("pid.New: " + err.Error())
	}
	return f
}

func equivalentSamples(tauSeconds float64, intervalMs int) int {
	if intervalMs <= 0 {
		return 1
	}
	return max(int(math.Round(tauSeconds*1000/float64(intervalMs))), 1)
}

// Compute runs one PID tick.
//   - dt: elapsed seconds since last tick
//   - measured: current process variable (e.g. % CPU)
//   - setpoint: desired target value
//
// Returns the new output clamped to [Min, Max].
func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	if math.IsNaN(measured) {
		return 0, errors.New("invalid measured metric")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dt, err := c.getDt(measureTime)
	if err != nil {
		return 0, err
	}

	spEff := c.refFilter.Filter(c.setpoint)
	c.refEffective = spEff

	e := spEff - measured

	p := c.Kp * e

	d := 0.0
	if !c.first && dt > 0 {
		d = c.Kd * (e - c.prevE) / dt
	}

	// Feedforward: sets the nominal operating point (u_ff = setpoint/K) so the
	// integral only trims model error / disturbances instead of ramping up.
	ff := 0.0
	if c.Kff > 0 {
		ff = spEff / c.Kff
	}

	// Integrate with conditional anti-windup: bound the integral to the band that
	// keeps the (unclamped) output off the saturation rails, and also to IClamp.
	// This stops windup during saturation (large setpoint steps, feedforward),
	// which is what caused the overshoot with a plain integral clamp.
	c.i += c.Ki * e * dt
	lo := math.Max(-c.IClamp, c.Min-ff-p-d)
	hi := math.Min(c.IClamp, c.Max-ff-p-d)
	if lo <= hi {
		c.i = clamp(c.i, lo, hi)
	} else {
		c.i = clamp(c.i, -c.IClamp, c.IClamp)
	}

	c.first = false
	c.prevE = e
	c.lastCompute = measureTime

	output := clamp(ff+p+c.i+d, c.Min, c.Max)

	metrics.Gauge(map[string]float64{
		"pid_setpoint": spEff,
		"pid_measured": measured,
		"pid_error":    e,
		"pid_p_term":   p,
		"pid_i_term":   c.i,
		"pid_d_term":   d,
		"pid_ff_term":  ff,
		"pid_output":   ff + p + c.i + d,
		"pid_kp":       c.Kp,
		"pid_ki":       c.Ki,
		"pid_kd":       c.Kd,
		"pid_i_clamp":  c.IClamp,
		"pid_max":      c.Max,
	})

	return output, nil
}

// GetParams returns a snapshot of the current PID parameters.
func (c *Controller) GetParams() entities.ControllerParams {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kp, ki, kd := c.Kp, c.Ki, c.Kd
	min, max, iclamp, setpoint := c.Min, c.Max, c.IClamp, c.setpoint
	spFilterType, spFilterSize := c.refFilter.Kind(), c.refFilter.Size()
	kff := c.Kff

	return entities.ControllerParams{
		Kp:                 &kp,
		Ki:                 &ki,
		Kd:                 &kd,
		Min:                &min,
		Max:                &max,
		IClamp:             &iclamp,
		Setpoint:           &setpoint,
		SetpointFilterType: &spFilterType,
		SetpointFilterSize: &spFilterSize,
		FfGain:             &kff,
	}
}

// SetParams updates PID parameters at runtime. Only non-nil fields are updated.
// Resets integral accumulator to avoid windup from previous gains.
func (c *Controller) SetParams(p entities.ControllerParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p.Kp != nil {
		c.Kp = *p.Kp
	}
	if p.Ki != nil {
		c.Ki = *p.Ki
	}
	if p.Kd != nil {
		c.Kd = *p.Kd
	}
	if p.Min != nil {
		c.Min = *p.Min
	}
	if p.Max != nil {
		c.Max = *p.Max
	}
	if p.IClamp != nil {
		c.IClamp = *p.IClamp
	}
	if p.Setpoint != nil {
		c.setpoint = *p.Setpoint
	}
	if p.FfGain != nil {
		c.Kff = *p.FfGain
	}

	if p.SetpointFilterType != nil || p.SetpointFilterSize != nil {
		kind, size := c.refFilter.Kind(), c.refFilter.Size()
		if p.SetpointFilterType != nil {
			kind = *p.SetpointFilterType
		}
		if p.SetpointFilterSize != nil {
			size = *p.SetpointFilterSize
		}
		f, err := filter.NewReferenceFilter(kind, size)
		if err != nil {
			return err
		}
		f.Seed(c.refEffective)
		c.refFilter = f
	}

	if c.Min > c.Max {
		return errors.New("pid: min must be <= max")
	}
	if c.IClamp < 0 {
		return errors.New("pid: i_clamp must be >= 0")
	}

	// Reset integral to avoid windup artifacts from previous gains
	c.i = 0

	return nil
}

func (c *Controller) getDt(measureTime time.Time) (float64, error) {
	if measureTime.Before(c.lastCompute) {
		return 0, errors.New("outdated measure")
	}
	dt := measureTime.Sub(c.lastCompute).Seconds()
	return dt, nil
}

func (c *Controller) SetSetpoint(setpoint float64) error {
	if math.IsNaN(setpoint) {
		return errors.New("invalid setpoint")
	}
	c.setpoint = setpoint
	return nil
}

func (c *Controller) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.i = 0
	c.prevE = 0
	c.first = true
	c.lastCompute = time.Time{}
	c.refFilter.Reset()
	c.refEffective = 0
}

func (c *Controller) Type() string { return "pid" }

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
