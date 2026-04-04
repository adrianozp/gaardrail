package controller_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/internal/controller"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// t0 is the zero time; adding durations to it gives predictable dt values.
var t0 = time.Time{}

func pid(kp, ki, kd, min, max, iClamp, setpoint float64) config.Config {
	return config.Config{PID: config.PID{Kp: kp, Ki: ki, Kd: kd, Min: min, Max: max, IClamp: iClamp, Setpoint: setpoint}}
}

func TestProportionalOnly(t *testing.T) {
	// Kp=1, Ki=0, Kd=0 → output = Kp * error = 1 * (15-5) = 10
	c := controller.New(pid(1.0, 0, 0, 0, 100, 20, 15.0))
	out, err := c.Compute(5.0, t0.Add(5*time.Second)) // dt=5s, measured=5, setpoint=15
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0, got %f", out)
	}
}

func TestOutputClamped(t *testing.T) {
	// Error=50, Kp=2 → raw output=100, but max=50
	c := controller.New(pid(2.0, 0, 0, 0.5, 50, 20, 50.0))
	out, err := c.Compute(0.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 50.0 {
		t.Errorf("expected 50.0 (clamped), got %f", out)
	}
}

func TestMinimumClamp(t *testing.T) {
	// CPU above setpoint → negative output → clamped to min=0.5
	c := controller.New(pid(1.0, 0, 0, 0.5, 50, 20, 15.0))
	out, err := c.Compute(30.0, t0.Add(5*time.Second)) // error = 15-30 = -15
	if err != nil {
		t.Fatal(err)
	}
	if out != 0.5 {
		t.Errorf("expected 0.5 (min clamp), got %f", out)
	}
}

func TestIntegralAntiWindup(t *testing.T) {
	// Ki=1, dt=5, error=5 for many ticks → I accumulates but is clamped to IClamp=20
	c := controller.New(pid(0, 1.0, 0, 0, 100, 20, 15.0))
	for i := 0; i < 20; i++ {
		// error=5, dt=5 each tick; unclamped I would be 500
		c.Compute(10.0, t0.Add(time.Duration(i+1)*5*time.Second))
	}
	// I should be clamped at 20, so output = I = 20
	out, err := c.Compute(15.0, t0.Add(21*5*time.Second)) // error=0, P=0, D=0 → output=I
	if err != nil {
		t.Fatal(err)
	}
	if out > 25 || out < 15 {
		t.Errorf("integral windup: expected ~20, got %f", out)
	}
}

func TestDerivative(t *testing.T) {
	// Kd=1, dt=5: error goes from 10 to 5 → D = Kd*(5-10)/5 = -1
	c := controller.New(pid(0, 0, 1.0, -100, 100, 20, 15.0))
	c.Compute(5.0, t0.Add(5*time.Second))               // first tick: error=10, D=0 (no prev)
	out, err := c.Compute(10.0, t0.Add(10*time.Second)) // second tick: error=5, D=(5-10)/5*Kd=-1
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-(-1.0)) > 0.001 {
		t.Errorf("expected D=-1.0, got %f", out)
	}
}

func TestReset(t *testing.T) {
	c := controller.New(pid(1.0, 1.0, 1.0, 0, 100, 20, 15.0))
	c.Compute(5.0, t0.Add(5*time.Second))
	c.Reset()
	out1, err := c.Compute(5.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	c2 := controller.New(pid(1.0, 1.0, 1.0, 0, 100, 20, 15.0))
	out2, err := c2.Compute(5.0, t0.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out1-out2) > 0.001 {
		t.Errorf("reset: expected same output as fresh controller, got %f vs %f", out1, out2)
	}
}

func TestSetSetpoint(t *testing.T) {
	c := controller.New(pid(1.0, 0, 0, 0, 100, 20, 10.0))
	if err := c.SetSetpoint(20.0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := c.Compute(10.0, t0.Add(5*time.Second)) // error = 20-10 = 10, P = 10
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(out-10.0) > 0.001 {
		t.Errorf("expected 10.0 after setpoint change, got %f", out)
	}
}

func TestSetSetpointNaN(t *testing.T) {
	c := controller.New(pid(1.0, 0, 0, 0, 100, 20, 10.0))
	if err := c.SetSetpoint(math.NaN()); err == nil {
		t.Error("expected error for NaN setpoint, got nil")
	}
}

func TestComputeNaNMeasured(t *testing.T) {
	c := controller.New(pid(1.0, 0, 0, 0, 100, 20, 15.0))
	_, err := c.Compute(math.NaN(), t0.Add(5*time.Second))
	if err == nil {
		t.Error("expected error for NaN measured, got nil")
	}
}

func TestComputeOutdatedTime(t *testing.T) {
	c := controller.New(pid(1.0, 0, 0, 0, 100, 20, 15.0))
	c.Compute(5.0, t0.Add(10*time.Second))
	_, err := c.Compute(5.0, t0.Add(5*time.Second)) // earlier than last compute
	if err == nil {
		t.Error("expected error for outdated measureTime, got nil")
	}
}

func TestNewPanicsMinGtMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for min > max")
		}
	}()
	controller.New(pid(1.0, 0, 0, 100, 50, 20, 15.0))
}

func TestNewPanicsNegativeIClamp(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for negative iClamp")
		}
	}()
	controller.New(pid(1.0, 0, 0, 0, 100, -1, 15.0))
}
