package control_test

import (
	"errors"
	"math"
	"testing"

	"github.com/adrianozp/gaardrail/pkg/control"
	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

var _ control.Controller = (*control.Smith)(nil)

func smithParams(kp, ki, lo, hi, iClamp, setpoint float64, m control.Model, sample float64) control.SmithParams {
	return control.SmithParams{
		Kp: kp, Ki: ki, Setpoint: setpoint, Model: m, SampleSeconds: sample,
		Limits: control.Limits{Min: lo, Max: hi, IClamp: iClamp},
	}
}

func noDelay() control.Model {
	return control.Model{K: 1, Tau: 1, Theta: 0}
}

func oneSampleDelay() control.Model {
	return control.Model{K: 2, Tau: 10, Theta: 10}
}

func newSmith(t *testing.T, p control.SmithParams) *control.Smith {
	t.Helper()
	c, err := control.NewSmith(p)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSmithProportionalOnlyWithoutDelay(t *testing.T) {
	out, _ := compute(t, newSmith(t, smithParams(1, 0, 0, 100, 20, 15, noDelay(), 1)), 5, 0)
	assertNear(t, out, 10)
}

func TestSmithOutputClamped(t *testing.T) {
	out, _ := compute(t, newSmith(t, smithParams(2, 0, 0.5, 50, 20, 50, noDelay(), 1)), 0, 0)
	assertNear(t, out, 50)
	out, _ = compute(t, newSmith(t, smithParams(1, 0, 0.5, 50, 20, 15, noDelay(), 1)), 30, 0)
	assertNear(t, out, 0.5)
}

func TestSmithFirstSampleDoesNotIntegrate(t *testing.T) {
	out, terms := compute(t, newSmith(t, smithParams(0, 1, 0, 100, 20, 15, noDelay(), 1)), 10, 0)
	assertNear(t, out, 0)
	assertNear(t, terms.I, 0)
}

func TestSmithIntegralAntiWindup(t *testing.T) {
	c := newSmith(t, smithParams(0, 1, 0, 100, 20, 15, noDelay(), 1))
	for i := range 20 {
		compute(t, c, 10, float64(i*5))
	}
	out, _ := compute(t, c, 15, 100)
	assertNear(t, out, 20)
}

func assertDeadTimeCompensated(t *testing.T, c *control.Smith) {
	t.Helper()
	out1, _ := compute(t, c, 0, 10)
	assertNear(t, out1, 100)
	out2, terms := compute(t, c, 0, 20)
	yModel := 2.0 * (1 - math.Exp(-1.0)) * 100.0
	assertNear(t, out2, 100-yModel)
	assertNear(t, terms.Prediction, yModel)
	assertNear(t, terms.Feedback, yModel)
}

func TestSmithDeadTimeCompensation(t *testing.T) {
	c := newSmith(t, smithParams(1, 0, -100, 100, 100, 100, oneSampleDelay(), 10))
	compute(t, c, 0, 0)
	c.Reset()
	assertDeadTimeCompensated(t, c)
}

func TestSmithSetParamsResizesDelay(t *testing.T) {
	c := newSmith(t, smithParams(1, 0, -100, 100, 100, 100, control.Model{K: 2, Tau: 10}, 10))
	p := c.Params()
	p.Model = oneSampleDelay()
	if err := c.SetParams(p); err != nil {
		t.Fatal(err)
	}
	assertDeadTimeCompensated(t, c)
}

func TestSmithResetMatchesFreshController(t *testing.T) {
	p := smithParams(1, 1, 0, 100, 20, 15, oneSampleDelay(), 10)
	c := newSmith(t, p)
	compute(t, c, 5, 10)
	compute(t, c, 5, 20)
	c.Reset()
	out1, _ := compute(t, c, 5, 10)
	out2, _ := compute(t, newSmith(t, p), 5, 10)
	assertNear(t, out1, out2)
}

func TestNewSmithRejectsInvalidModel(t *testing.T) {
	bad := []control.SmithParams{
		smithParams(1, 0, 0, 100, 20, 50, oneSampleDelay(), 0),
		smithParams(1, 0, 0, 100, 20, 50, control.Model{K: 1, Tau: -1}, 10),
		smithParams(1, 0, 0, 100, 20, 50, control.Model{K: 1, Tau: 1, Theta: -1}, 10),
	}
	for _, p := range bad {
		if _, err := control.NewSmith(p); !errors.Is(err, control.ErrInvalidModel) {
			t.Errorf("%+v: got %v", p.Model, err)
		}
	}
}

func TestNewSmithRejectsInvalidLimits(t *testing.T) {
	if _, err := control.NewSmith(smithParams(1, 0, 100, 50, 20, 15, noDelay(), 1)); !errors.Is(err, control.ErrInvalidLimits) {
		t.Errorf("min > max: got %v", err)
	}
	if _, err := control.NewSmith(smithParams(1, 0, 0, 100, -1, 15, noDelay(), 1)); !errors.Is(err, control.ErrNegativeIClamp) {
		t.Errorf("negative i_clamp: got %v", err)
	}
}

func TestSmithRejectedSetParamsKeepsPreviousParams(t *testing.T) {
	c := newSmith(t, smithParams(1, 0, 0, 100, 20, 50, oneSampleDelay(), 10))
	before := c.Params()
	if err := c.SetParams(smithParams(5, 0, 0, 100, 20, 50, oneSampleDelay(), 0)); err == nil {
		t.Fatal("expected error for sample_seconds=0")
	}
	if c.Params() != before {
		t.Fatalf("params changed after a rejected update: %+v", c.Params())
	}
}

func TestSmithSetParamsResetsIntegral(t *testing.T) {
	c := newSmith(t, smithParams(0, 1, 0, 100, 20, 15, noDelay(), 1))
	compute(t, c, 10, 0)
	compute(t, c, 10, 5)
	p := c.Params()
	p.Kp = 2
	if err := c.SetParams(p); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, c, 15, 10)
	assertNear(t, out, 0)
}

func TestSmithSetSetpoint(t *testing.T) {
	c := newSmith(t, smithParams(1, 0, 0, 100, 20, 10, noDelay(), 1))
	if err := c.SetSetpoint(20); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, c, 10, 0)
	assertNear(t, out, 10)
	if err := c.SetSetpoint(math.NaN()); !errors.Is(err, control.ErrInvalidSetpoint) {
		t.Errorf("NaN setpoint: got %v", err)
	}
}

func TestSmithRejectsBadMeasurements(t *testing.T) {
	c := newSmith(t, smithParams(1, 0, 0, 100, 20, 15, noDelay(), 1))
	if _, _, err := c.Compute(math.NaN(), at(0)); !errors.Is(err, control.ErrInvalidMeasurement) {
		t.Errorf("NaN measured: got %v", err)
	}
	compute(t, c, 5, 10)
	if _, _, err := c.Compute(5, at(5)); !errors.Is(err, control.ErrOutdatedMeasurement) {
		t.Errorf("outdated: got %v", err)
	}
}

func TestSmithExponentialSetpointFilterRamps(t *testing.T) {
	p := smithParams(1, 0, -100, 100, 100, 50, noDelay(), 1)
	p.SetpointFilter = filter.Spec{Kind: filter.Exponential, Size: 2}
	out, _ := compute(t, newSmith(t, p), 0, 0)
	assertNear(t, out, (1-math.Exp(-0.5))*50)
}

func TestSmithSetParamsSwitchesFilterFromCurrentEffectiveReference(t *testing.T) {
	p := smithParams(1, 0, -100, 100, 100, 100, noDelay(), 1)
	p.SetpointFilter = filter.Spec{Kind: filter.MovingAverage, Size: 4}
	c := newSmith(t, p)
	compute(t, c, 0, 0)
	p.SetpointFilter = filter.Spec{Kind: filter.Exponential, Size: 4}
	if err := c.SetParams(p); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, c, 0, 1)
	a := math.Exp(-1.0 / 4.0)
	assertNear(t, out, a*25+(1-a)*100)
}
