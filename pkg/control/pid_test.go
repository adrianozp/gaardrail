package control_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/pkg/control"
	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

var (
	_  control.Controller = (*control.PID)(nil)
	t0                    = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

func at(seconds float64) time.Time {
	return t0.Add(time.Duration(seconds * float64(time.Second)))
}

func pidParams(kp, ki, kd, lo, hi, iClamp, setpoint float64) control.PIDParams {
	return control.PIDParams{
		Kp: kp, Ki: ki, Kd: kd, Setpoint: setpoint,
		Limits: control.Limits{Min: lo, Max: hi, IClamp: iClamp},
	}
}

func newPID(t *testing.T, p control.PIDParams) *control.PID {
	t.Helper()
	c, err := control.NewPID(p)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func compute(t *testing.T, c control.Controller, measured, seconds float64) (float64, control.Terms) {
	t.Helper()
	out, terms, err := c.Compute(measured, at(seconds))
	if err != nil {
		t.Fatal(err)
	}
	return out, terms
}

func assertNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPIDProportionalOnly(t *testing.T) {
	out, _ := compute(t, newPID(t, pidParams(1, 0, 0, 0, 100, 20, 15)), 5, 0)
	assertNear(t, out, 10)
}

func TestPIDOutputClampedToMax(t *testing.T) {
	out, _ := compute(t, newPID(t, pidParams(2, 0, 0, 0.5, 50, 20, 50)), 0, 0)
	assertNear(t, out, 50)
}

func TestPIDOutputClampedToMin(t *testing.T) {
	out, _ := compute(t, newPID(t, pidParams(1, 0, 0, 0.5, 50, 20, 15)), 30, 0)
	assertNear(t, out, 0.5)
}

func TestPIDFirstSampleDoesNotIntegrate(t *testing.T) {
	c := newPID(t, pidParams(0, 1, 0, 0, 100, 20, 15))
	out, terms := compute(t, c, 10, 0)
	assertNear(t, out, 0)
	assertNear(t, terms.I, 0)
}

func TestPIDSecondSampleIntegratesElapsedTime(t *testing.T) {
	c := newPID(t, pidParams(0, 1, 0, 0, 100, 20, 15))
	compute(t, c, 10, 0)
	out, _ := compute(t, c, 10, 2)
	assertNear(t, out, 10)
}

func TestPIDIntegralAntiWindup(t *testing.T) {
	c := newPID(t, pidParams(0, 1, 0, 0, 100, 20, 15))
	for i := range 20 {
		compute(t, c, 10, float64(i*5))
	}
	out, _ := compute(t, c, 15, 100)
	assertNear(t, out, 20)
}

func TestPIDDerivative(t *testing.T) {
	c := newPID(t, pidParams(0, 0, 1, -100, 100, 20, 15))
	_, first := compute(t, c, 5, 0)
	assertNear(t, first.D, 0)
	out, _ := compute(t, c, 10, 5)
	assertNear(t, out, -1)
}

func TestPIDFeedforward(t *testing.T) {
	p := pidParams(0, 0, 0, 0, 100, 20, 50)
	p.FFGain = 2.5
	out, terms := compute(t, newPID(t, p), 0, 0)
	assertNear(t, out, 20)
	assertNear(t, terms.FF, 20)
}

func TestPIDTermsDescribeTheTick(t *testing.T) {
	c := newPID(t, pidParams(2, 1, 0, 0, 100, 50, 15))
	compute(t, c, 10, 0)
	_, terms := compute(t, c, 10, 1)
	want := control.Terms{Setpoint: 15, Measured: 10, Error: 5, P: 10, I: 5, Output: 15}
	if terms != want {
		t.Fatalf("got %+v, want %+v", terms, want)
	}
}

func TestPIDResetMatchesFreshController(t *testing.T) {
	c := newPID(t, pidParams(1, 1, 1, 0, 100, 20, 15))
	compute(t, c, 5, 0)
	compute(t, c, 7, 5)
	c.Reset()
	out1, _ := compute(t, c, 5, 0)
	out2, _ := compute(t, newPID(t, pidParams(1, 1, 1, 0, 100, 20, 15)), 5, 0)
	assertNear(t, out1, out2)
}

func TestPIDSetSetpoint(t *testing.T) {
	c := newPID(t, pidParams(1, 0, 0, 0, 100, 20, 10))
	if err := c.SetSetpoint(20); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, c, 10, 0)
	assertNear(t, out, 10)
	assertNear(t, c.Params().Setpoint, 20)
}

func TestPIDRejectsNonFiniteInputs(t *testing.T) {
	c := newPID(t, pidParams(1, 0, 0, 0, 100, 20, 10))
	if err := c.SetSetpoint(math.NaN()); !errors.Is(err, control.ErrInvalidSetpoint) {
		t.Errorf("NaN setpoint: got %v", err)
	}
	if _, _, err := c.Compute(math.NaN(), at(0)); !errors.Is(err, control.ErrInvalidMeasurement) {
		t.Errorf("NaN measured: got %v", err)
	}
	if _, _, err := c.Compute(math.Inf(1), at(0)); !errors.Is(err, control.ErrInvalidMeasurement) {
		t.Errorf("Inf measured: got %v", err)
	}
}

func TestPIDRejectsOutdatedMeasurement(t *testing.T) {
	c := newPID(t, pidParams(1, 0, 0, 0, 100, 20, 15))
	compute(t, c, 5, 10)
	if _, _, err := c.Compute(5, at(5)); !errors.Is(err, control.ErrOutdatedMeasurement) {
		t.Fatalf("got %v", err)
	}
}

func TestNewPIDRejectsInvalidParams(t *testing.T) {
	if _, err := control.NewPID(pidParams(1, 0, 0, 100, 50, 20, 15)); !errors.Is(err, control.ErrInvalidLimits) {
		t.Errorf("min > max: got %v", err)
	}
	if _, err := control.NewPID(pidParams(1, 0, 0, 0, 100, -1, 15)); !errors.Is(err, control.ErrNegativeIClamp) {
		t.Errorf("negative i_clamp: got %v", err)
	}
	p := pidParams(1, 0, 0, 0, 100, 20, 15)
	p.SetpointFilter = filter.Spec{Kind: "median"}
	if _, err := control.NewPID(p); err == nil {
		t.Error("unknown filter: expected error")
	}
}

func TestPIDRejectedSetParamsKeepsPreviousParams(t *testing.T) {
	c := newPID(t, pidParams(1, 0, 0, 0, 100, 20, 15))
	before := c.Params()
	if err := c.SetParams(pidParams(9, 0, 0, 500, 100, 20, 15)); err == nil {
		t.Fatal("expected error for min > max")
	}
	if c.Params() != before {
		t.Fatalf("params changed after a rejected update: %+v", c.Params())
	}
}

func TestPIDSetParamsResetsIntegral(t *testing.T) {
	c := newPID(t, pidParams(0, 1, 0, 0, 100, 20, 15))
	compute(t, c, 10, 0)
	compute(t, c, 10, 5)
	if err := c.SetParams(pidParams(0, 1, 0, 0, 100, 20, 15)); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, c, 15, 10)
	assertNear(t, out, 0)
}
