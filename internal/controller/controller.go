package controller

import (
	"errors"
	"math"
	"time"

	"github.com/adrianozp/gaardrail/pkg/config"
)

// Controller is a discrete PID controller in position form.
// output(t) = clamp(P + I(t) + D, Min, Max)
// This is the absolute output (e.g. drain rate RPS), not a delta.
type Controller struct {
	Kp, Ki, Kd float64
	Min, Max   float64 // output clamp
	// IClamp is the anti-windup bound for the integral term.
	// Should be <= Max for correct saturation behavior.
	// If IClamp > Max, the integral alone can saturate the output at Max
	// while continuing to grow, which undermines the anti-windup guarantee.
	IClamp float64

	setpoint    float64
	i           float64
	prevE       float64
	first       bool
	lastCompute time.Time
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
		Kp:       cfg.PID.Kp,
		Ki:       cfg.PID.Ki,
		Kd:       cfg.PID.Kd,
		Min:      cfg.PID.Min,
		Max:      cfg.PID.Max,
		IClamp:   cfg.PID.IClamp,
		first:    true,
		setpoint: cfg.PID.Setpoint,
	}
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
	dt, err := c.getDt(measureTime)
	if err != nil {
		return 0, err
	}

	e := c.setpoint - measured

	p := c.Kp * e
	c.i = clamp(c.i+c.Ki*e*dt, -c.IClamp, c.IClamp)

	d := 0.0
	if !c.first && dt > 0 {
		d = c.Kd * (e - c.prevE) / dt
	}
	c.first = false
	c.prevE = e
	c.lastCompute = measureTime

	return clamp(p+c.i+d, c.Min, c.Max), nil
}

func (c Controller) getDt(measureTime time.Time) (float64, error) {
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
	c.i = 0
	c.prevE = 0
	c.first = true
	c.lastCompute = time.Time{}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
