package control

import (
	"errors"
	"math"
	"time"

	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

var (
	ErrInvalidMeasurement  = errors.New("control: measured value must be a finite number")
	ErrInvalidSetpoint     = errors.New("control: setpoint must be a finite number")
	ErrOutdatedMeasurement = errors.New("control: measurement is older than the previous one")
	ErrInvalidLimits       = errors.New("control: min must be <= max")
	ErrNegativeIClamp      = errors.New("control: i_clamp must be >= 0")
	ErrInvalidModel        = errors.New("control: model tau and sample period must be > 0 and theta >= 0")
	ErrInvalidOutput       = errors.New("control: output must be a finite number")
)

type Terms struct {
	Setpoint   float64
	Measured   float64
	Error      float64
	P          float64
	I          float64
	D          float64
	FF         float64
	Output     float64
	Feedback   float64
	Prediction float64
}

type Controller interface {
	Compute(measured float64, at time.Time) (float64, Terms, error)
	SetSetpoint(setpoint float64) error
	Reset()
}

type Limits struct {
	Min    float64
	Max    float64
	IClamp float64
}

func (l Limits) validate() error {
	if l.Min > l.Max {
		return ErrInvalidLimits
	}
	if l.IClamp < 0 {
		return ErrNegativeIClamp
	}
	return nil
}

func (l Limits) clamp(v float64) float64 {
	return clamp(v, l.Min, l.Max)
}

func (l Limits) integrate(integral, delta, otherTerms float64) float64 {
	lo := math.Max(-l.IClamp, l.Min-otherTerms)
	hi := math.Min(l.IClamp, l.Max-otherTerms)
	if lo > hi {
		lo, hi = -l.IClamp, l.IClamp
	}
	return clamp(integral+delta, lo, hi)
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func checkMeasurement(measured float64) error {
	if !isFinite(measured) {
		return ErrInvalidMeasurement
	}
	return nil
}

func checkSetpoint(setpoint float64) error {
	if !isFinite(setpoint) {
		return ErrInvalidSetpoint
	}
	return nil
}

type sampleClock struct {
	last    time.Time
	started bool
}

func (c *sampleClock) advance(at time.Time) (float64, error) {
	if !c.started {
		c.last, c.started = at, true
		return 0, nil
	}
	if at.Before(c.last) {
		return 0, ErrOutdatedMeasurement
	}
	dt := at.Sub(c.last).Seconds()
	c.last = at
	return dt, nil
}

type reference struct {
	filter    *filter.Signal
	effective float64
}

func newReference(spec filter.Spec) (reference, error) {
	f, err := filter.NewReference(spec)
	return reference{filter: f}, err
}

func (r *reference) next(setpoint float64) float64 {
	r.effective = r.filter.Filter(setpoint)
	return r.effective
}

func (r *reference) retune(spec filter.Spec) error {
	if spec == r.filter.Spec() {
		return nil
	}
	f, err := filter.NewReference(spec)
	if err != nil {
		return err
	}
	f.Seed(r.effective)
	r.filter = f
	return nil
}

func (r *reference) reset() {
	r.filter.Reset()
	r.effective = 0
}
