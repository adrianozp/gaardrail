// Package smith implements a discrete Smith predictor controller.
//
// It wraps a PI controller with an internal FOPDT model of the plant
//
//	G(s) = K * e^{-theta s} / (tau s + 1)
//
// so the PI can act on a delay-free prediction of the process output and thus
// compensate the transport delay (metric scrape lag + irate window + sampling),
// which is the dominant limit on disturbance rejection in this system.
//
// The feedback signal fed to the PI is
//
//	yFeedback = measured + (yModel - yModelDelayed)
//
// where yModel is the delay-free model response to the control output and
// yModelDelayed is that same response delayed by round(theta/T) samples. When
// the model matches the plant, (measured - yModelDelayed) ~= 0 and the PI sees
// the delay-free prediction; the correction term keeps it robust to mismatch.
//
// The output is the absolute drain rate (RPS), clamped to [Min, Max]; it is not
// a delta. Kd is not used (PI only): with metric noise and large dead time a
// derivative term is counterproductive.
package smith

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

// Controller is a discrete Smith predictor (PI + internal FOPDT model).
type Controller struct {
	mu       sync.RWMutex
	Kp, Ki   float64
	Min, Max float64 // output clamp
	IClamp   float64 // anti-windup bound for the integral term

	// Internal model parameters.
	modelK, modelTau, modelTheta float64
	sampleSeconds                float64

	setpoint    float64
	i           float64 // integral accumulator
	yModel      float64 // delay-free model output (state of the first-order model)
	prevU       float64 // last control output applied to the model
	delay       []float64
	delayHead   int
	first       bool
	lastCompute time.Time
	filter      *filter.MovingAverage
}

// New builds a Smith predictor from config. Panics on invalid static config
// (min > max, negative iClamp, non-positive model tau), matching pid.New.
func New(cfg config.Config) *Controller {
	s := cfg.Smith
	if s.Min > s.Max {
		panic("smith.New: min must be <= max")
	}
	if s.IClamp < 0 {
		panic("smith.New: iClamp must be >= 0")
	}
	// Model params come from identification; when unset (zero-value config) fall
	// back to safe values instead of panicking, so a bare Config stays usable.
	modelTau := s.ModelTau
	if modelTau <= 0 {
		modelTau = 1.0
	}
	sampleSeconds := s.SampleSeconds
	if sampleSeconds <= 0 {
		sampleSeconds = 1.0
	}

	c := &Controller{
		Kp:            s.Kp,
		Ki:            s.Ki,
		Min:           s.Min,
		Max:           s.Max,
		IClamp:        s.IClamp,
		modelK:        s.ModelK,
		modelTau:      modelTau,
		modelTheta:    s.ModelTheta,
		sampleSeconds: sampleSeconds,
		setpoint:      s.Setpoint,
		first:         true,
		filter:        filter.NewMovingAverage(s.FilterSize),
	}
	c.resizeDelay()
	return c
}

// resizeDelay (re)allocates the dead-time ring buffer from the current model
// theta and sample period. Caller must hold the write lock (or be in New).
func (c *Controller) resizeDelay() {
	d := max(int(math.Round(c.modelTheta/c.sampleSeconds)), 0)
	c.delay = make([]float64, d)
	c.delayHead = 0
}

// Compute runs one Smith predictor tick and returns the drain rate clamped to
// [Min, Max]. measureTime is used to derive the true elapsed dt (same contract
// as the PID controller: out-of-order samples are rejected).
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

	// Smooth the (noisy) measurement before it enters the predictor feedback.
	measured = c.filter.Filter(measured)

	// Advance the delay-free first-order model with the previous control output
	// held over dt (ZOH): yModel <- a*yModel + K*(1-a)*prevU, a = e^{-dt/tau}.
	a := math.Exp(-dt / c.modelTau)
	c.yModel = a*c.yModel + c.modelK*(1-a)*c.prevU

	// Delayed model output (theta seconds ago) via the ring buffer.
	delayed := c.yModel
	if len(c.delay) > 0 {
		delayed = c.delay[c.delayHead]
		c.delay[c.delayHead] = c.yModel
		c.delayHead = (c.delayHead + 1) % len(c.delay)
	}

	// Smith predictor feedback: measured + (delay-free model - delayed model).
	feedback := measured + (c.yModel - delayed)
	e := c.setpoint - feedback

	p := c.Kp * e

	// Integrate with conditional anti-windup (same scheme as the PID): bound the
	// integral to the band that keeps the output off the rails, and to IClamp.
	c.i += c.Ki * e * dt
	lo := math.Max(-c.IClamp, c.Min-p)
	hi := math.Min(c.IClamp, c.Max-p)
	if lo <= hi {
		c.i = clamp(c.i, lo, hi)
	} else {
		c.i = clamp(c.i, -c.IClamp, c.IClamp)
	}

	output := clamp(p+c.i, c.Min, c.Max)

	c.prevU = output
	c.first = false
	c.lastCompute = measureTime

	metrics.Gauge(map[string]float64{
		"smith_setpoint":   c.setpoint,
		"smith_error":      e,
		"smith_feedback":   feedback,
		"smith_prediction": c.yModel,
		"smith_p_term":     p,
		"smith_i_term":     c.i,
		"smith_output":     output,
		"smith_kp":         c.Kp,
		"smith_ki":         c.Ki,
		"smith_i_clamp":    c.IClamp,
		"smith_max":        c.Max,
	})

	return output, nil
}

// GetParams returns a snapshot of the current PI parameters plus the internal
// FOPDT model (model params are read-only: set via config, shown for insight).
func (c *Controller) GetParams() entities.ControllerParams {
	c.mu.RLock()
	defer c.mu.RUnlock()

	kp, ki := c.Kp, c.Ki
	min, max, iclamp, setpoint := c.Min, c.Max, c.IClamp, c.setpoint
	mk, mtau, mtheta, ss := c.modelK, c.modelTau, c.modelTheta, c.sampleSeconds
	fsize := c.filter.Size()

	return entities.ControllerParams{
		Kp:            &kp,
		Ki:            &ki,
		Min:           &min,
		Max:           &max,
		IClamp:        &iclamp,
		Setpoint:      &setpoint,
		FilterSize:    &fsize,
		ModelK:        &mk,
		ModelTau:      &mtau,
		ModelTheta:    &mtheta,
		SampleSeconds: &ss,
	}
}

// SetParams updates PI parameters at runtime. Only non-nil fields are applied.
// Kd is ignored (PI only). Resets the integral to avoid windup from prior gains.
func (c *Controller) SetParams(p entities.ControllerParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p.Kp != nil {
		c.Kp = *p.Kp
	}
	if p.Ki != nil {
		c.Ki = *p.Ki
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

	// Internal FOPDT model (from identification): allows runtime re-tuning of the
	// predictor without a restart. Theta or sample changes resize the dead-time
	// ring buffer (d = round(theta/T)) and reset the model state.
	resizeModel := false
	if p.ModelK != nil {
		c.modelK = *p.ModelK
	}
	if p.ModelTau != nil {
		if *p.ModelTau <= 0 {
			return errors.New("smith: model_tau must be > 0")
		}
		c.modelTau = *p.ModelTau
	}
	if p.ModelTheta != nil {
		if *p.ModelTheta < 0 {
			return errors.New("smith: model_theta must be >= 0")
		}
		c.modelTheta = *p.ModelTheta
		resizeModel = true
	}
	if p.SampleSeconds != nil {
		if *p.SampleSeconds <= 0 {
			return errors.New("smith: sample_seconds must be > 0")
		}
		c.sampleSeconds = *p.SampleSeconds
		resizeModel = true
	}

	if c.Min > c.Max {
		return errors.New("smith: min must be <= max")
	}
	if c.IClamp < 0 {
		return errors.New("smith: i_clamp must be >= 0")
	}

	if resizeModel {
		c.resizeDelay()
		c.yModel = 0
		c.prevU = 0
		c.first = true
	}

	c.i = 0
	return nil
}

func (c *Controller) getDt(measureTime time.Time) (float64, error) {
	if measureTime.Before(c.lastCompute) {
		return 0, errors.New("outdated measure")
	}
	return measureTime.Sub(c.lastCompute).Seconds(), nil
}

func (c *Controller) SetSetpoint(setpoint float64) error {
	if math.IsNaN(setpoint) {
		return errors.New("invalid setpoint")
	}
	c.mu.Lock()
	c.setpoint = setpoint
	c.mu.Unlock()
	return nil
}

// Reset clears the integral, the internal model state and the dead-time buffer.
func (c *Controller) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.i = 0
	c.yModel = 0
	c.prevU = 0
	c.first = true
	c.lastCompute = time.Time{}
	c.filter.Reset()
	c.resizeDelay()
}

func (c *Controller) Type() string { return "smith" }

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
