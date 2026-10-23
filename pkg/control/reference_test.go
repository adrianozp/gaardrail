package control_test

import (
	"math"
	"testing"

	"github.com/adrianozp/gaardrail/pkg/control"
	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

func filteredPID(t *testing.T, setpoint float64, spec filter.Spec) *control.PID {
	t.Helper()
	p := pidParams(1, 0, 0, -100, 100, 100, setpoint)
	p.SetpointFilter = spec
	return newPID(t, p)
}

func withFilter(p control.PIDParams, spec filter.Spec) control.PIDParams {
	p.SetpointFilter = spec
	return p
}

func TestSetpointMovingAverageRampsReference(t *testing.T) {
	c := filteredPID(t, 50, filter.Spec{Kind: filter.MovingAverage, Size: 4})
	out, terms := compute(t, c, 0, 0)
	assertNear(t, out, 12.5)
	assertNear(t, terms.Setpoint, 12.5)
}

func TestSetParamsSwitchesFilterFromCurrentEffectiveReference(t *testing.T) {
	c := filteredPID(t, 100, filter.Spec{Kind: filter.MovingAverage, Size: 4})
	out0, _ := compute(t, c, 0, 0)
	assertNear(t, out0, 25)
	spec := filter.Spec{Kind: filter.Exponential, Size: 4}
	if err := c.SetParams(withFilter(c.Params(), spec)); err != nil {
		t.Fatal(err)
	}
	out1, _ := compute(t, c, 0, 1)
	a := math.Exp(-1.0 / 4.0)
	assertNear(t, out1, a*25+(1-a)*100)
}

func TestSetParamsKeepsFilterStateWhenSpecUnchanged(t *testing.T) {
	c := filteredPID(t, 100, filter.Spec{Kind: filter.MovingAverage, Size: 4})
	compute(t, c, 0, 0)
	p := c.Params()
	p.Kp = 1
	if err := c.SetParams(p); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, c, 0, 1)
	assertNear(t, out, 50)
}

func TestSetParamsInvalidFilterKeepsOldFilter(t *testing.T) {
	c := filteredPID(t, 100, filter.Spec{Kind: filter.MovingAverage, Size: 4})
	compute(t, c, 0, 0)
	if err := c.SetParams(withFilter(c.Params(), filter.Spec{Kind: "bogus"})); err == nil {
		t.Fatal("expected error for an unknown filter type")
	}
	out, _ := compute(t, c, 0, 1)
	assertNear(t, out, 50)
}

func TestParamsReportNormalizedFilter(t *testing.T) {
	c := newPID(t, pidParams(1, 0, 0, 0, 100, 20, 15))
	if got := c.Params().SetpointFilter; got != (filter.Spec{Kind: filter.None, Size: 1}) {
		t.Fatalf("got %+v", got)
	}
}
