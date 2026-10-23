package control

import (
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

type PIDParams struct {
	Kp             float64
	Ki             float64
	Kd             float64
	FFGain         float64
	Setpoint       float64
	SetpointFilter filter.Spec
	Limits
}

func (p PIDParams) normalize() (PIDParams, error) {
	if err := p.validate(); err != nil {
		return PIDParams{}, err
	}
	if err := checkSetpoint(p.Setpoint); err != nil {
		return PIDParams{}, err
	}
	spec, err := p.SetpointFilter.Normalize()
	p.SetpointFilter = spec
	return p, err
}

type PID struct {
	mu        sync.Mutex
	params    PIDParams
	ref       reference
	clock     sampleClock
	integral  float64
	prevError float64
}

func NewPID(p PIDParams) (*PID, error) {
	p, err := p.normalize()
	if err != nil {
		return nil, err
	}
	ref, err := newReference(p.SetpointFilter)
	if err != nil {
		return nil, err
	}
	return &PID{params: p, ref: ref}, nil
}

func (c *PID) Compute(measured float64, at time.Time) (float64, Terms, error) {
	if err := checkMeasurement(measured); err != nil {
		return 0, Terms{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	dt, err := c.clock.advance(at)
	if err != nil {
		return 0, Terms{}, err
	}
	t := c.step(measured, dt)
	return t.Output, t, nil
}

func (c *PID) step(measured, dt float64) Terms {
	sp := c.ref.next(c.params.Setpoint)
	e := sp - measured
	t := Terms{Setpoint: sp, Measured: measured, Error: e, P: c.params.Kp * e}
	t.D = c.derivative(e, dt)
	t.FF = c.feedforward(sp)
	c.integral = c.params.integrate(c.integral, c.params.Ki*e*dt, t.FF+t.P+t.D)
	t.I = c.integral
	t.Output = c.params.clamp(t.FF + t.P + t.I + t.D)
	c.prevError = e
	return t
}

func (c *PID) derivative(e, dt float64) float64 {
	if dt <= 0 {
		return 0
	}
	return c.params.Kd * (e - c.prevError) / dt
}

func (c *PID) feedforward(setpoint float64) float64 {
	if c.params.FFGain <= 0 {
		return 0
	}
	return setpoint / c.params.FFGain
}

func (c *PID) Params() PIDParams {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.params
}

func (c *PID) SetParams(p PIDParams) error {
	p, err := p.normalize()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ref.retune(p.SetpointFilter); err != nil {
		return err
	}
	c.params = p
	c.integral = 0
	return nil
}

func (c *PID) SetSetpoint(setpoint float64) error {
	if err := checkSetpoint(setpoint); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.params.Setpoint = setpoint
	return nil
}

func (c *PID) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.integral = 0
	c.prevError = 0
	c.clock = sampleClock{}
	c.ref.reset()
}
