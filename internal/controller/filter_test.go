package controller_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/internal/controller"
)

func TestMovingAveragePassthrough(t *testing.T) {
	// size=1: filter is a passthrough — output equals input exactly
	ma := controller.NewMovingAverage(1)
	if got := ma.Filter(42.0); math.Abs(got-42.0) > 0.001 {
		t.Errorf("expected 42.0, got %f", got)
	}
	if got := ma.Filter(7.5); math.Abs(got-7.5) > 0.001 {
		t.Errorf("expected 7.5, got %f", got)
	}
}

func TestMovingAveragePartialWindow(t *testing.T) {
	// size=3, only 2 values added: average of those 2
	ma := controller.NewMovingAverage(3)
	ma.Filter(10.0)
	got := ma.Filter(20.0)
	// avg(10, 20) = 15
	if math.Abs(got-15.0) > 0.001 {
		t.Errorf("expected 15.0, got %f", got)
	}
}

func TestMovingAverageFullWindow(t *testing.T) {
	// size=3, three values: average of all three
	ma := controller.NewMovingAverage(3)
	ma.Filter(10.0)
	ma.Filter(20.0)
	got := ma.Filter(30.0)
	// avg(10, 20, 30) = 20
	if math.Abs(got-20.0) > 0.001 {
		t.Errorf("expected 20.0, got %f", got)
	}
}

func TestMovingAverageCircularBuffer(t *testing.T) {
	// size=3, fourth value drops the oldest
	ma := controller.NewMovingAverage(3)
	ma.Filter(10.0) // window: [10]
	ma.Filter(20.0) // window: [10, 20]
	ma.Filter(30.0) // window: [10, 20, 30]
	got := ma.Filter(40.0) // window: [20, 30, 40], drops 10
	// avg(20, 30, 40) = 30
	if math.Abs(got-30.0) > 0.001 {
		t.Errorf("expected 30.0 (oldest dropped), got %f", got)
	}
}

func TestMovingAveragePanicsOnZeroSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for size=0")
		}
	}()
	controller.NewMovingAverage(0)
}

func TestComputeWithMovingAverageFilter(t *testing.T) {
	// Kp=1, Ki=0, Kd=0, setpoint=10, filter size=3
	// Feed three measurements: 0, 0, 6
	// filtered measured = avg(0, 0, 6) = 2
	// error = 10 - 2 = 8 → output = 8
	cfg := pid(1.0, 0, 0, -100, 100, 20, 10.0)
	cfg.PID.FilterSize = 3
	c := controller.New(cfg)

	c.Compute(0.0, t0.Add(1*time.Second))
	c.Compute(0.0, t0.Add(2*time.Second))
	out, err := c.Compute(6.0, t0.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	// expected: P = Kp * (setpoint - filtered) = 1 * (10 - 2) = 8
	if math.Abs(out-8.0) > 0.001 {
		t.Errorf("expected 8.0 (filtered), got %f", out)
	}
}
