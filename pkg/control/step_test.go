package control_test

import (
	"errors"
	"math"
	"testing"

	"github.com/adrianozp/gaardrail/pkg/control"
)

var _ control.Controller = (*control.Step)(nil)

func newStep(t *testing.T, output float64) *control.Step {
	t.Helper()
	s, err := control.NewStep(output)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStepOutputsConstant(t *testing.T) {
	out, terms := compute(t, newStep(t, 5), 42, 0)
	assertNear(t, out, 5)
	if terms != (control.Terms{Measured: 42, Output: 5}) {
		t.Fatalf("got %+v", terms)
	}
}

func TestStepSetOutput(t *testing.T) {
	s := newStep(t, 5)
	if err := s.SetOutput(9); err != nil {
		t.Fatal(err)
	}
	out, _ := compute(t, s, 0, 0)
	assertNear(t, out, 9)
	assertNear(t, s.Output(), 9)
}

func TestStepRejectsNonFiniteOutput(t *testing.T) {
	if _, err := control.NewStep(math.NaN()); !errors.Is(err, control.ErrInvalidOutput) {
		t.Errorf("NewStep: got %v", err)
	}
	if err := newStep(t, 5).SetOutput(math.Inf(1)); !errors.Is(err, control.ErrInvalidOutput) {
		t.Errorf("SetOutput: got %v", err)
	}
}
