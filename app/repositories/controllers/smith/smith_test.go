package smith_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/smith"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// t0 is the zero time; adding durations to it gives predictable dt values.
var t0 = time.Time{}

// cfg builds a Smith config. Use mtheta=0 to make the predictor collapse to a
// plain PI controller acting directly on the measured value.
func cfg(kp, ki, min, max, iClamp, setpoint, mk, mtau, mtheta, sample float64) config.Config {
	return config.Config{Smith: config.Smith{
		Kp: kp, Ki: ki, Min: min, Max: max, IClamp: iClamp, Setpoint: setpoint,
		ModelK: mk, ModelTau: mtau, ModelTheta: mtheta, SampleSeconds: sample,
	}}
}

// With no dead time (theta=0) the predictor correction is zero, so the
// controller reduces to a PI on the measured value: out = Kp*error.
func TestProportionalOnlyNoDelay(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0, 100, 20, 15.0, 1, 1, 0, 1))
	out, err := c.Compute(5.0, t0.Add(5*time.Second)) // error = 15-5 = 10
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0, got %f", out)
	}
}

func TestOutputClamped(t *testing.T) {
	c := smith.New(cfg(2.0, 0, 0.5, 50, 20, 50.0, 1, 1, 0, 1))
	out, err := c.Compute(0.0, t0.Add(5*time.Second)) // raw 100, clamped to 50
	if err != nil {
		t.Fatal(err)
	}
	if out != 50.0 {
		t.Errorf("expected 50.0 (clamped), got %f", out)
	}
}

func TestMinimumClamp(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0.5, 50, 20, 15.0, 1, 1, 0, 1))
	out, err := c.Compute(30.0, t0.Add(5*time.Second)) // error negative → min clamp
	if err != nil {
		t.Fatal(err)
	}
	if out != 0.5 {
		t.Errorf("expected 0.5 (min clamp), got %f", out)
	}
}

func TestIntegralAntiWindup(t *testing.T) {
	c := smith.New(cfg(0, 1.0, 0, 100, 20, 15.0, 1, 1, 0, 1))
	for i := 0; i < 20; i++ {
		c.Compute(10.0, t0.Add(time.Duration(i+1)*5*time.Second)) // error=5, dt=5
	}
	out, err := c.Compute(15.0, t0.Add(21*5*time.Second)) // error=0 → output = I
	if err != nil {
		t.Fatal(err)
	}
	if out > 25 || out < 15 {
		t.Errorf("integral windup: expected ~20, got %f", out)
	}
}

// TestDeadTimeCompensation is the Smith-specific property: on the second tick
// the internal model already predicts the (delayed) plant rise, so the PI pulls
// back before the delayed measurement reflects it. A plain PI on the measured
// value would still command +Max here.
func TestDeadTimeCompensation(t *testing.T) {
	// modelK=2, tau=10, theta=10, T=10 → dead-time buffer of 1 sample.
	c := smith.New(cfg(1.0, 0, -100, 100, 100, 100.0, 2, 10, 10, 10))

	// Tick 1: prevU=0 so model output is 0; feedback=measured=0; out=Kp*100=100.
	out1, err := c.Compute(0.0, t0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out1-100.0) > 0.001 {
		t.Fatalf("tick1: expected 100, got %f", out1)
	}

	// Tick 2: model advances with prevU=100. Measured is still 0 (delayed), but
	// the delay-free model predicts yModel = K*(1-e^{-1})*100, so the predictor
	// feedback reverses the control action.
	out2, err := c.Compute(0.0, t0.Add(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	a := math.Exp(-1.0)
	yModel := 2.0 * (1 - a) * 100.0
	want := 100.0 - yModel // e = setpoint - feedback, output = Kp*e
	if math.Abs(out2-want) > 0.01 {
		t.Errorf("tick2: expected dead-time-compensated output %f, got %f", want, out2)
	}
	if out2 >= 0 {
		t.Errorf("tick2: predictor should have reversed the action (out<0), got %f", out2)
	}
}

func TestReset(t *testing.T) {
	c := smith.New(cfg(1.0, 1.0, 0, 100, 20, 15.0, 2, 10, 10, 10))
	c.Compute(5.0, t0.Add(10*time.Second))
	c.Compute(5.0, t0.Add(20*time.Second))
	c.Reset()
	out1, err := c.Compute(5.0, t0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	fresh := smith.New(cfg(1.0, 1.0, 0, 100, 20, 15.0, 2, 10, 10, 10))
	out2, err := fresh.Compute(5.0, t0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out1-out2) > 0.001 {
		t.Errorf("reset: expected same output as fresh controller, got %f vs %f", out1, out2)
	}
}

func TestSetParamsUpdatesModelAndResizesDelay(t *testing.T) {
	// Start with NO dead time (theta=0 -> delay buffer depth 0): a plain PI.
	c := smith.New(cfg(1.0, 0, -100, 100, 100, 100.0, 2, 10, 0, 10))

	// Re-tune the internal model at runtime to theta=10, T=10 -> depth 1,
	// matching TestDeadTimeCompensation. This must resize the ring buffer.
	f := func(v float64) *float64 { return &v }
	if err := c.SetParams(entities.ControllerParams{
		ModelK: f(2), ModelTau: f(10), ModelTheta: f(10), SampleSeconds: f(10),
	}); err != nil {
		t.Fatal(err)
	}

	// Same two-tick sequence as TestDeadTimeCompensation: the predictor must now
	// compensate the (runtime-added) dead time and reverse the action on tick 2.
	if _, err := c.Compute(0.0, t0.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	out2, err := c.Compute(0.0, t0.Add(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	want := 100.0 - 2.0*(1-math.Exp(-1.0))*100.0
	if math.Abs(out2-want) > 0.01 {
		t.Errorf("runtime model update: expected dead-time-compensated %f, got %f", want, out2)
	}
	if out2 >= 0 {
		t.Errorf("expected predictor to reverse the action (out<0), got %f", out2)
	}
}

func TestSetParamsRejectsInvalidModel(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0, 100, 20, 50.0, 2, 10, 10, 10))
	f := func(v float64) *float64 { return &v }
	if err := c.SetParams(entities.ControllerParams{SampleSeconds: f(0)}); err == nil {
		t.Error("expected error for sample_seconds=0")
	}
	if err := c.SetParams(entities.ControllerParams{ModelTau: f(-1)}); err == nil {
		t.Error("expected error for model_tau<0")
	}
}

func TestSetParamsResetsIntegral(t *testing.T) {
	c := smith.New(cfg(0, 1.0, 0, 100, 20, 15.0, 1, 1, 0, 1))
	c.Compute(10.0, t0.Add(5*time.Second)) // build up some integral
	kp := 2.0
	if err := c.SetParams(entityKp(kp)); err != nil {
		t.Fatal(err)
	}
	// After SetParams the integral is reset; with error=0 the output is 0.
	out, err := c.Compute(15.0, t0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out) > 0.001 {
		t.Errorf("expected integral reset (output 0), got %f", out)
	}
}

func TestSetSetpoint(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0, 100, 20, 10.0, 1, 1, 0, 1))
	if err := c.SetSetpoint(20.0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := c.Compute(10.0, t0.Add(5*time.Second)) // error = 20-10 = 10
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0 after setpoint change, got %f", out)
	}
}

func TestSetSetpointNaN(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0, 100, 20, 10.0, 1, 1, 0, 1))
	if err := c.SetSetpoint(math.NaN()); err == nil {
		t.Error("expected error for NaN setpoint, got nil")
	}
}

func TestComputeNaNMeasured(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0, 100, 20, 15.0, 1, 1, 0, 1))
	if _, err := c.Compute(math.NaN(), t0.Add(5*time.Second)); err == nil {
		t.Error("expected error for NaN measured, got nil")
	}
}

func TestComputeOutdatedTime(t *testing.T) {
	c := smith.New(cfg(1.0, 0, 0, 100, 20, 15.0, 1, 1, 0, 1))
	c.Compute(5.0, t0.Add(10*time.Second))
	if _, err := c.Compute(5.0, t0.Add(5*time.Second)); err == nil {
		t.Error("expected error for outdated measureTime, got nil")
	}
}

func TestNewPanicsMinGtMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for min > max")
		}
	}()
	smith.New(cfg(1.0, 0, 100, 50, 20, 15.0, 1, 1, 0, 1))
}

func TestNewPanicsNegativeIClamp(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative iClamp")
		}
	}()
	smith.New(cfg(1.0, 0, 0, 100, -1, 15.0, 1, 1, 0, 1))
}

// Unset model params (zero-value config) must not panic: they fall back to safe
// defaults so a bare Config (e.g. before identification) stays usable.
func TestNewToleratesUnsetModelParams(t *testing.T) {
	c := smith.New(config.Config{Smith: config.Smith{Kp: 1.0, Max: 100, IClamp: 20, Setpoint: 15.0}})
	out, err := c.Compute(5.0, t0.Add(1*time.Second)) // theta=0 fallback → PI on measured
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0, got %f", out)
	}
}

// entityKp builds a ControllerParams patch that sets only Kp.
func entityKp(kp float64) entities.ControllerParams {
	return entities.ControllerParams{Kp: &kp}
}

func TestSmithSemFiltroDeSetpointMantemDegrau(t *testing.T) {
	sCfg := config.Config{Smith: config.Smith{Kp: 1, Max: 100, Min: -100, IClamp: 100, Setpoint: 50}}
	c := smith.New(sCfg)
	base := time.Now()
	out, err := c.Compute(0, base.Add(1*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-50) > 1e-6 {
		t.Fatalf("sem filtro de setpoint, degrau deve ser cru: got %v", out)
	}
}

func TestSmithComFiltroExponencialRampa(t *testing.T) {
	sCfg := config.Config{Smith: config.Smith{Kp: 1, Max: 100, Min: -100, IClamp: 100,
		Setpoint: 50, SetpointFilterType: "exponential", SetpointFilterSize: 2}}
	c := smith.New(sCfg)
	base := time.Now()
	out, err := c.Compute(0, base.Add(1*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	want := (1 - math.Exp(-0.5)) * 50
	if math.Abs(out-want) > 1e-6 {
		t.Fatalf("1º tique com filtro exponencial size=2: got %v want %v", out, want)
	}
}
