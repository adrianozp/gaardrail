package autopid

import (
	"math"
	"testing"
)

func approx(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %.6g, want %.6g (tol %.3g)", name, got, want, tol)
	}
}

// syntheticStep samples a noise-free FOPDT step response:
//
//	y(t) = y0 + du*K*(1 - e^{-(t-theta)/tau}) for t >= theta, else y0.
func syntheticStep(y0, du, k, tau, theta, dt, tEnd float64) []sample {
	var curve []sample
	for t := 0.0; t <= tEnd; t += dt {
		y := y0
		if t >= theta {
			y = y0 + du*k*(1-math.Exp(-(t-theta)/tau))
		}
		curve = append(curve, sample{T: t, Y: y})
	}
	return curve
}

func TestIdentifyFOPDT_RecoversModel(t *testing.T) {
	const (
		y0    = 50.0
		du    = 10.0
		k     = 2.0
		tau   = 30.0
		theta = 10.0
	)
	curve := syntheticStep(y0, du, k, tau, theta, 1.0, 300.0)

	m, err := identifyFOPDT(y0, curve, du)
	if err != nil {
		t.Fatalf("identifyFOPDT: %v", err)
	}
	// Two-point method recovers a true FOPDT within a couple percent.
	approx(t, "K", m.K, k, 0.05)
	approx(t, "Tau", m.Tau, tau, 0.02*tau)
	approx(t, "Theta", m.Theta, theta, 0.02*tau)
}

func TestIdentifyFOPDT_Degenerate(t *testing.T) {
	curve := syntheticStep(50, 10, 2, 30, 10, 1.0, 300.0)
	if _, err := identifyFOPDT(50, curve, 0); err == nil {
		t.Error("expected error for zero input step")
	}
	flat := []sample{{T: 0, Y: 50}, {T: 100, Y: 50}}
	if _, err := identifyFOPDT(50, flat, 10); err == nil {
		t.Error("expected error for no measurable response")
	}
}

func TestCrossTimeInterpolates(t *testing.T) {
	curve := []sample{{T: 0, Y: 0}, {T: 10, Y: 10}, {T: 20, Y: 20}}
	tc, ok := crossTime(curve, 5, 1)
	if !ok {
		t.Fatal("level not crossed")
	}
	approx(t, "cross", tc, 5, 1e-9)
}

func TestSIMCPI(t *testing.T) {
	// K=2, tau=30, theta=10, tauC=0 => tauC=theta=10.
	g := simcPI(2, 30, 10, 0)
	approx(t, "Kp", g.Kp, 0.75, 1e-9)
	approx(t, "Ki", g.Ki, 0.025, 1e-9)
	approx(t, "Kd", g.Kd, 0, 1e-12)
}

func TestAMIGOPI(t *testing.T) {
	// K=2, tau=30, theta=10 (hand-computed).
	g := amigoPI(2, 30, 10)
	approx(t, "Kp", g.Kp, 0.31875, 1e-6)
	approx(t, "Ki", g.Ki, 0.31875/26.0, 1e-9)
	approx(t, "Kd", g.Kd, 0, 1e-12)
}

func TestAMIGOPID(t *testing.T) {
	// K=2, tau=30, theta=10 (hand-computed).
	g := amigoPID(2, 30, 10)
	approx(t, "Kp", g.Kp, 0.775, 1e-9)
	approx(t, "Ki", g.Ki, 0.775*13.0/280.0, 1e-9)
	approx(t, "Kd", g.Kd, 0.775*(150.0/33.0), 1e-9)
}

func TestComputeGainsDispatch(t *testing.T) {
	m := fopdt{K: 2, Tau: 30, Theta: 10}
	if _, err := computeGains(m, 10, "amigo", "pi", 0); err != nil {
		t.Errorf("amigo pi: %v", err)
	}
	if _, err := computeGains(m, 10, "simc", "pi", 5); err != nil {
		t.Errorf("simc: %v", err)
	}
	if _, err := computeGains(m, 10, "bogus", "pi", 0); err == nil {
		t.Error("expected error for unknown rule")
	}
	if _, err := computeGains(m, 0, "amigo", "pi", 0); err == nil {
		t.Error("expected error for non-positive thetaEff")
	}
}
