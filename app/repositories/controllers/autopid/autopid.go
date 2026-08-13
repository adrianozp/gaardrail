// Package autopid implements a self-tuning (auto-PID) controller.
//
// On activation it runs an open-loop step experiment on the drain rate, records
// the CPU reaction curve, identifies a FOPDT model of the plant
//
//	G(s) = K e^{-theta s} / (tau s + 1)
//
// by the two-point method of Smith (28.3%/63.2%), computes the PID gains
// automatically via the configured tuning rule (AMIGO or SIMC) and then closes
// the loop with an embedded pid.Controller using those gains. It therefore
// removes the manual offline tuning step (the same procedure as identify.m in
// the gaardrail-flood-test repo's matlab tooling).
//
// The controller advances through three phases, driven by the measurement clock:
//
//	BASELINE -> STEP (identify) -> CONTROL
//
// BASELINE holds a low drain rate to establish the output baseline y0; STEP
// applies a higher drain rate and records the reaction curve; at the end of STEP
// the model is identified, the gains are computed and applied, and the embedded
// PID takes over. Reset() returns to BASELINE and re-identifies on the next run,
// which is what switchable does when this controller is (re)activated.
package autopid

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/pid"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/rs/zerolog/log"
)

type phase int

const (
	phaseBaseline phase = iota
	phaseStep
	phaseControl
)

// Controller is a self-tuning controller wrapping an embedded pid.Controller.
type Controller struct {
	mu sync.RWMutex

	// Embedded control-phase engine (filter, anti-windup, setpoint filter). Its
	// gains are overwritten with the identified tuning at the end of STEP.
	pid *pid.Controller

	// Experiment/tuning configuration resolved from config.
	baselineOutput  float64
	stepOutput      float64
	baselineSeconds float64
	identifySeconds float64
	tauC            float64
	rule            string
	mode            string

	// Runtime experiment state.
	ph              phase
	started         bool      // true once the first Compute of this run has been seen
	startTime       time.Time // measure time of the first Compute of this run
	lastCompute     time.Time
	stepStart       time.Time
	baselineSamples []float64
	curve           []sample

	// Identification results (set at the STEP->CONTROL transition).
	tuned bool
	model fopdt
	g     gains
}

// New builds the auto-PID controller. The embedded PID inherits the operating
// limits (min/max/i_clamp/setpoint/filter) from cfg.PID; its gains are replaced
// by the auto-tuned values once identification completes.
func New(cfg config.Config) *Controller {
	ap := cfg.AutoPID

	baseline := ap.BaselineOutput
	if baseline == 0 {
		baseline = cfg.PID.Min
	}
	step := ap.StepOutput
	if step == 0 {
		step = cfg.PID.Max
	}
	rule := ap.TuningRule
	if rule == "" {
		rule = "amigo"
	}
	mode := ap.Mode
	if mode == "" {
		mode = "pi"
	}

	c := &Controller{
		pid:             pid.New(cfg),
		baselineOutput:  baseline,
		stepOutput:      step,
		baselineSeconds: ap.BaselineSeconds,
		identifySeconds: ap.IdentifySeconds,
		tauC:            ap.TauC,
		rule:            rule,
		mode:            mode,
		ph:              phaseBaseline,
	}
	return c
}

// Compute advances the experiment/control state machine one tick and returns the
// drain rate. During BASELINE/STEP the output is the fixed experiment level; in
// CONTROL it delegates to the embedded PID.
func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	if math.IsNaN(measured) {
		return 0, errors.New("autopid: invalid measured metric")
	}

	c.mu.Lock()

	if !c.started {
		c.started = true
		c.startTime = measureTime
		c.lastCompute = measureTime
	}
	if measureTime.Before(c.lastCompute) {
		c.mu.Unlock()
		return 0, errors.New("autopid: outdated measure")
	}
	c.lastCompute = measureTime

	switch c.ph {
	case phaseBaseline:
		c.baselineSamples = append(c.baselineSamples, measured)
		if measureTime.Sub(c.startTime).Seconds() >= c.baselineSeconds {
			c.ph = phaseStep
			c.stepStart = measureTime
		}
		out := c.baselineOutput
		c.emit()
		c.mu.Unlock()
		return out, nil

	case phaseStep:
		tRel := measureTime.Sub(c.stepStart).Seconds()
		c.curve = append(c.curve, sample{T: tRel, Y: measured})
		if tRel >= c.identifySeconds {
			c.tune()
			c.mu.Unlock()
			return c.pid.Compute(measured, measureTime)
		}
		out := c.stepOutput
		c.emit()
		c.mu.Unlock()
		return out, nil

	default: // phaseControl
		c.emit()
		c.mu.Unlock()
		return c.pid.Compute(measured, measureTime)
	}
}

// tune identifies the FOPDT model, computes the gains and applies them to the
// embedded PID, then switches to CONTROL. On any failure it keeps the PID's
// configured default gains and logs a warning. Caller must hold the write lock.
func (c *Controller) tune() {
	c.ph = phaseControl

	y0 := mean(c.baselineSamples)
	du := c.stepOutput - c.baselineOutput

	model, err := identifyFOPDT(y0, c.curve, du)
	if err != nil {
		log.Warn().Err(err).Msg("autopid: identification failed, keeping default PID gains")
		return
	}

	// Effective dead time seen by the discrete loop: identified theta plus one
	// sampling period T (ZOH + sampling).
	thetaEff := model.Theta + medianDt(c.curve)

	g, err := computeGains(model, thetaEff, c.rule, c.mode, c.tauC)
	if err != nil {
		log.Warn().Err(err).Msg("autopid: tuning failed, keeping default PID gains")
		return
	}

	if err := c.pid.SetParams(entities.ControllerParams{Kp: &g.Kp, Ki: &g.Ki, Kd: &g.Kd}); err != nil {
		log.Warn().Err(err).Msg("autopid: applying tuned gains failed, keeping default PID gains")
		return
	}

	c.model = model
	c.g = g
	c.tuned = true
	log.Info().
		Float64("K", model.K).Float64("tau", model.Tau).Float64("theta", model.Theta).
		Float64("kp", g.Kp).Float64("ki", g.Ki).Float64("kd", g.Kd).
		Str("rule", c.rule).Str("mode", c.mode).
		Msg("autopid: identified plant and applied auto-tuned gains")
}

// emit publishes the experiment phase and (once tuned) the identified model and
// gains. During CONTROL the embedded PID also publishes its own pid_* gauges.
func (c *Controller) emit() {
	m := map[string]float64{"autopid_phase": float64(c.ph)}
	if c.tuned {
		m["autopid_k"] = c.model.K
		m["autopid_tau"] = c.model.Tau
		m["autopid_theta"] = c.model.Theta
		m["autopid_kp"] = c.g.Kp
		m["autopid_ki"] = c.g.Ki
		m["autopid_kd"] = c.g.Kd
	}
	metrics.Gauge(m)
}

// GetParams returns the embedded PID snapshot (gains + operating limits),
// augmented with the identified FOPDT model once identification has run.
func (c *Controller) GetParams() entities.ControllerParams {
	p := c.pid.GetParams()

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.tuned {
		k, tau, theta := c.model.K, c.model.Tau, c.model.Theta
		p.ModelK = &k
		p.ModelTau = &tau
		p.ModelTheta = &theta
	}
	return p
}

// SetParams forwards to the embedded PID so gains/limits can be adjusted at
// runtime (e.g. after auto-tuning). Only non-nil fields are applied.
func (c *Controller) SetParams(p entities.ControllerParams) error {
	return c.pid.SetParams(p)
}

// SetSetpoint forwards to the embedded PID; the control phase tracks it.
func (c *Controller) SetSetpoint(setpoint float64) error {
	return c.pid.SetSetpoint(setpoint)
}

// Reset returns the controller to BASELINE and clears identification state, so
// the next run re-identifies the plant. The embedded PID is also reset.
func (c *Controller) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ph = phaseBaseline
	c.started = false
	c.startTime = time.Time{}
	c.lastCompute = time.Time{}
	c.stepStart = time.Time{}
	c.baselineSamples = nil
	c.curve = nil
	c.tuned = false
	c.model = fopdt{}
	c.g = gains{}
	c.pid.Reset()
}

func (c *Controller) Type() string { return "autopid" }

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// medianDt returns the median sampling period of the reaction curve, used as the
// effective sampling period T. Falls back to 1s for a degenerate curve.
func medianDt(curve []sample) float64 {
	if len(curve) < 2 {
		return 1.0
	}
	diffs := make([]float64, 0, len(curve)-1)
	for i := 1; i < len(curve); i++ {
		if d := curve[i].T - curve[i-1].T; d > 0 {
			diffs = append(diffs, d)
		}
	}
	if len(diffs) == 0 {
		return 1.0
	}
	sortFloats(diffs)
	return diffs[len(diffs)/2]
}

// sortFloats is a small insertion sort (curves are short: a few dozen samples).
func sortFloats(xs []float64) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j-1] > xs[j]; j-- {
			xs[j-1], xs[j] = xs[j], xs[j-1]
		}
	}
}
