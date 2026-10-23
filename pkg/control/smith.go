package control

import (
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

type SmithParams struct {
	Kp             float64
	Ki             float64
	Setpoint       float64
	SetpointFilter filter.Spec
	Model          Model
	SampleSeconds  float64
	Limits
}

func (p SmithParams) normalize() (SmithParams, error) {
	if err := p.validate(); err != nil {
		return SmithParams{}, err
	}
	if err := checkSetpoint(p.Setpoint); err != nil {
		return SmithParams{}, err
	}
	if err := p.Model.validate(p.SampleSeconds); err != nil {
		return SmithParams{}, err
	}
	spec, err := p.SetpointFilter.Normalize()
	p.SetpointFilter = spec
	return p, err
}

type Smith struct {
	mu        sync.Mutex
	params    SmithParams
	ref       reference
	clock     sampleClock
	integral  float64
	predictor *predictor
}

func NewSmith(p SmithParams) (*Smith, error) {
	p, err := p.normalize()
	if err != nil {
		return nil, err
	}
	ref, err := newReference(p.SetpointFilter)
	if err != nil {
		return nil, err
	}
	return &Smith{params: p, ref: ref, predictor: newPredictor(p.Model, p.SampleSeconds)}, nil
}

func (c *Smith) Compute(measured float64, at time.Time) (float64, Terms, error) {
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

func (c *Smith) step(measured, dt float64) Terms {
	sp := c.ref.next(c.params.Setpoint)
	now, delayed := c.predictor.advance(dt)
	feedback := measured + now - delayed
	e := sp - feedback
	t := Terms{Setpoint: sp, Measured: measured, Error: e, P: c.params.Kp * e, Feedback: feedback, Prediction: now}
	c.integral = c.params.integrate(c.integral, c.params.Ki*e*dt, t.P)
	t.I = c.integral
	t.Output = c.params.clamp(t.P + t.I)
	c.predictor.hold(t.Output)
	return t
}

func (c *Smith) Params() SmithParams {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.params
}

func (c *Smith) SetParams(p SmithParams) error {
	p, err := p.normalize()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ref.retune(p.SetpointFilter); err != nil {
		return err
	}
	c.retunePredictor(p)
	c.params = p
	c.integral = 0
	return nil
}

func (c *Smith) retunePredictor(p SmithParams) {
	if p.Model.Theta != c.params.Model.Theta || p.SampleSeconds != c.params.SampleSeconds {
		c.predictor = newPredictor(p.Model, p.SampleSeconds)
		return
	}
	c.predictor.model = p.Model
}

func (c *Smith) SetSetpoint(setpoint float64) error {
	if err := checkSetpoint(setpoint); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.params.Setpoint = setpoint
	return nil
}

func (c *Smith) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.integral = 0
	c.clock = sampleClock{}
	c.ref.reset()
	c.predictor = newPredictor(c.params.Model, c.params.SampleSeconds)
}
