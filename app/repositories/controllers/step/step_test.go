package step_test

import (
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/step"
	"github.com/adrianozp/gaardrail/pkg/config"
)

var t0 = time.Time{}

func cfg(max float64) config.Config {
	return config.Config{PID: config.PID{Max: max}}
}

func ptr(v float64) *float64 { return &v }

func TestStepOutputsMax(t *testing.T) {
	c := step.NewStep(cfg(5.0))
	out, err := c.Compute(42.0, t0.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 5.0 {
		t.Errorf("expected 5.0, got %f", out)
	}
}

func TestStepGetParamsReflectsMax(t *testing.T) {
	c := step.NewStep(cfg(5.0))
	p := c.GetParams()
	if p.Max == nil {
		t.Fatal("expected Max to be set, got nil")
	}
	if *p.Max != 5.0 {
		t.Errorf("expected Max 5.0, got %f", *p.Max)
	}
}

func TestStepSetParamsNilSafe(t *testing.T) {
	c := step.NewStep(cfg(5.0))
	// PATCH bodies omit unchanged fields, so every pointer may be nil.
	if err := c.SetParams(entities.ControllerParams{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := c.GetParams()
	if p.Max == nil || *p.Max != 5.0 {
		t.Errorf("max should be unchanged at 5.0, got %v", p.Max)
	}
}

func TestStepSetParamsUpdatesMax(t *testing.T) {
	c := step.NewStep(cfg(5.0))
	if err := c.SetParams(entities.ControllerParams{Max: ptr(9.0)}); err != nil {
		t.Fatal(err)
	}
	out, err := c.Compute(0, t0.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 9.0 {
		t.Errorf("expected output 9.0 after update, got %f", out)
	}
}
