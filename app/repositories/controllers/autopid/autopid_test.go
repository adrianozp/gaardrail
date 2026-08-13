package autopid_test

import (
	"math"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/autopid"
	"github.com/adrianozp/gaardrail/pkg/config"
)

var _ controllers.Controller = (*autopid.Controller)(nil)

var t0 = time.Time{}

func testCfg() config.Config {
	return config.Config{
		PID: config.PID{Min: 0, Max: 100, IClamp: 100, Setpoint: 70},
		AutoPID: config.AutoPID{
			TuningRule:      "amigo",
			Mode:            "pi",
			BaselineOutput:  2,
			StepOutput:      12,
			BaselineSeconds: 60,
			IdentifySeconds: 300,
		},
	}
}

// fopdtResponse is the plant used to drive identification in tests.
func fopdtResponse(y0, du, k, tau, theta, tStep float64) float64 {
	if tStep < theta {
		return y0
	}
	return y0 + du*k*(1-math.Exp(-(tStep-theta)/tau))
}

func TestPhaseOutputs(t *testing.T) {
	c := autopid.New(testCfg())

	// BASELINE: elapsed < 60s -> baseline_output (2).
	out, err := c.Compute(50, t0.Add(0*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 2 {
		t.Errorf("baseline output: got %v, want 2", out)
	}
	out, _ = c.Compute(50, t0.Add(30*time.Second))
	if out != 2 {
		t.Errorf("baseline output mid: got %v, want 2", out)
	}

	// The tick that reaches 60s still returns baseline and arms the step.
	out, _ = c.Compute(50, t0.Add(60*time.Second))
	if out != 2 {
		t.Errorf("baseline boundary: got %v, want 2", out)
	}

	// STEP: now the output is step_output (12).
	out, _ = c.Compute(55, t0.Add(70*time.Second))
	if out != 12 {
		t.Errorf("step output: got %v, want 12", out)
	}
}

func TestFullCycleTunesAndControls(t *testing.T) {
	const (
		y0    = 50.0
		du    = 10.0 // step_output - baseline_output = 12 - 2
		k     = 2.0
		tau   = 30.0
		theta = 10.0
	)
	c := autopid.New(testCfg())

	var lastOut float64
	for sec := 0; sec <= 360; sec += 10 {
		mt := t0.Add(time.Duration(sec) * time.Second)
		var measured float64
		if sec <= 60 {
			measured = y0
		} else {
			measured = fopdtResponse(y0, du, k, tau, theta, float64(sec-60))
		}
		out, err := c.Compute(measured, mt)
		if err != nil {
			t.Fatalf("Compute at %ds: %v", sec, err)
		}
		lastOut = out
	}

	// After the step window the model must be identified and exposed.
	p := c.GetParams()
	if p.ModelK == nil || p.ModelTau == nil || p.ModelTheta == nil {
		t.Fatal("expected identified model in params after tuning")
	}
	if math.Abs(*p.ModelK-k) > 0.1 {
		t.Errorf("ModelK: got %v, want ~%v", *p.ModelK, k)
	}
	if math.Abs(*p.ModelTau-tau) > 0.15*tau {
		t.Errorf("ModelTau: got %v, want ~%v", *p.ModelTau, tau)
	}
	if math.Abs(*p.ModelTheta-theta) > 0.15*tau {
		t.Errorf("ModelTheta: got %v, want ~%v", *p.ModelTheta, theta)
	}

	// Auto-tuned gains must be positive and non-trivial.
	if p.Kp == nil || *p.Kp <= 0 {
		t.Errorf("expected positive tuned Kp, got %v", p.Kp)
	}
	if p.Ki == nil || *p.Ki <= 0 {
		t.Errorf("expected positive tuned Ki, got %v", p.Ki)
	}

	// The control-phase output must stay within the operating limits.
	if lastOut < 0 || lastOut > 100 {
		t.Errorf("control output out of range: %v", lastOut)
	}
}

func TestControlConvergesToSetpoint(t *testing.T) {
	// After tuning, drive the embedded PID with a static-gain plant (y = K*u)
	// and check the CPU converges near the setpoint.
	const k = 2.0
	c := autopid.New(testCfg())

	// Run the experiment phase to completion (identification correctness is
	// covered elsewhere; here we only need it to reach CONTROL).
	for sec := 0; sec <= 360; sec += 10 {
		mt := t0.Add(time.Duration(sec) * time.Second)
		measured := 50.0
		if sec > 60 {
			measured = 50 + 10*k*(1-math.Exp(-(float64(sec-60)-10)/30))
		}
		if _, err := c.Compute(measured, mt); err != nil {
			t.Fatalf("warmup Compute: %v", err)
		}
	}

	// Closed loop: y_next = clamp(K*u, 0, 100). Setpoint is 70.
	measured := 60.0
	sec := 370
	for range 400 {
		mt := t0.Add(time.Duration(sec) * time.Second)
		u, err := c.Compute(measured, mt)
		if err != nil {
			t.Fatalf("control Compute: %v", err)
		}
		measured = math.Max(0, math.Min(100, k*u))
		sec += 10
	}
	if math.Abs(measured-70) > 3 {
		t.Errorf("did not converge to setpoint: got %v, want ~70", measured)
	}
}

func TestOutOfOrderAndNaN(t *testing.T) {
	c := autopid.New(testCfg())
	if _, err := c.Compute(math.NaN(), t0.Add(10*time.Second)); err == nil {
		t.Error("expected error for NaN measurement")
	}
	if _, err := c.Compute(50, t0.Add(50*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Compute(50, t0.Add(40*time.Second)); err == nil {
		t.Error("expected error for out-of-order measure")
	}
}

func TestResetReidentifies(t *testing.T) {
	c := autopid.New(testCfg())
	// Advance into the step phase.
	_, _ = c.Compute(50, t0.Add(0*time.Second))
	_, _ = c.Compute(50, t0.Add(70*time.Second))

	c.Reset()

	// After reset we are back in BASELINE: first output is baseline_output again.
	out, err := c.Compute(50, t0.Add(0*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out != 2 {
		t.Errorf("after reset expected baseline output 2, got %v", out)
	}
}
