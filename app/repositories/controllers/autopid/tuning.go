package autopid

import (
	"errors"
	"math"
)

// fopdt is a first-order plus dead time plant model:
//
//	G(s) = K * e^{-Theta s} / (Tau s + 1).
type fopdt struct {
	K     float64 // static gain (%CPU per msg/s)
	Tau   float64 // time constant (s)
	Theta float64 // transport delay (s), continuous
}

// gains holds PID gains in position form, matching pid.Controller:
// Ki = Kp/Ti and Kd = Kp*Td (Kd = 0 for PI).
type gains struct {
	Kp, Ki, Kd float64
}

// sample is one point of the step reaction curve: T seconds after the step
// started, the (filtered) process value was Y.
type sample struct {
	T float64
	Y float64
}

// tailFraction is the portion of the reaction curve (by time) averaged to
// estimate the steady-state value yinf, mirroring identify.m.
const tailFraction = 0.1

// identifyFOPDT estimates a FOPDT model from an open-loop step response using the
// two-point method of Smith (28.3% / 63.2%), the same procedure as
// matlab/identify.m. y0 is the pre-step baseline; curve is the reaction curve
// (times relative to the step, ascending); du is the input step amplitude
// (stepOutput - baselineOutput). Returns an error when the experiment is
// degenerate (no input step, no measurable response, or non-positive tau).
func identifyFOPDT(y0 float64, curve []sample, du float64) (fopdt, error) {
	if math.Abs(du) < 1e-9 {
		return fopdt{}, errors.New("autopid: no input step (du ~= 0)")
	}
	if len(curve) < 2 {
		return fopdt{}, errors.New("autopid: not enough samples to identify")
	}

	// yinf: mean of the final tail of the curve.
	tEnd := curve[len(curve)-1].T
	tStart := curve[0].T
	tailStart := tEnd - tailFraction*(tEnd-tStart)
	var sum float64
	var n int
	for _, s := range curve {
		if s.T >= tailStart {
			sum += s.Y
			n++
		}
	}
	if n == 0 {
		return fopdt{}, errors.New("autopid: empty reaction tail")
	}
	yinf := sum / float64(n)

	dy := yinf - y0
	if math.Abs(dy) < 1e-9 {
		return fopdt{}, errors.New("autopid: no measurable response (dy ~= 0)")
	}

	k := dy / du

	t28, ok28 := crossTime(curve, y0+0.283*dy, dy)
	t63, ok63 := crossTime(curve, y0+0.632*dy, dy)
	if !ok28 || !ok63 {
		return fopdt{}, errors.New("autopid: reaction curve never crossed the 28%/63% levels")
	}

	tau := 1.5 * (t63 - t28)
	if tau <= 0 {
		return fopdt{}, errors.New("autopid: non-positive tau from two-point method")
	}
	theta := math.Max(t63-tau, 0)

	return fopdt{K: k, Tau: tau, Theta: theta}, nil
}

// crossTime returns the first time the curve crosses level, with linear
// interpolation between samples. dy sets the direction (rising if dy >= 0).
// The second return is false if the level is never reached. Ported from the
// crossTime helper in identify.m.
func crossTime(curve []sample, level, dy float64) (float64, bool) {
	rising := dy >= 0
	for i, s := range curve {
		crossed := (rising && s.Y >= level) || (!rising && s.Y <= level)
		if !crossed {
			continue
		}
		if i == 0 {
			return s.T, true
		}
		p := curve[i-1]
		if s.Y == p.Y {
			return s.T, true
		}
		// linear interpolation between p and s
		return p.T + (level-p.Y)*(s.T-p.T)/(s.Y-p.Y), true
	}
	return 0, false
}

// computeGains dispatches to the selected tuning rule. thetaEff is the effective
// dead time seen by the loop (theta + one sampling period T, for the ZOH +
// sampling delay). tauC is the SIMC closed-loop time constant (<=0 => thetaEff).
func computeGains(m fopdt, thetaEff float64, rule, mode string, tauC float64) (gains, error) {
	if m.K == 0 {
		return gains{}, errors.New("autopid: zero plant gain")
	}
	if thetaEff <= 0 {
		return gains{}, errors.New("autopid: non-positive effective dead time")
	}
	switch rule {
	case "simc":
		return simcPI(m.K, m.Tau, thetaEff, tauC), nil
	case "amigo":
		if mode == "pid" {
			return amigoPID(m.K, m.Tau, thetaEff), nil
		}
		return amigoPI(m.K, m.Tau, thetaEff), nil
	default:
		return gains{}, errors.New("autopid: unknown tuning rule " + rule)
	}
}

// simcPI is the SIMC/Skogestad PI rule for a FOPDT plant, in position form.
// Ported from simcPI in identify.m: Kc = (1/K)*tau/(tauc+theta),
// tauI = min(tau, 4*(tauc+theta)). tauC <= 0 uses the robust default tauc=theta.
func simcPI(k, tau, theta, tauC float64) gains {
	if tauC <= 0 {
		tauC = theta
	}
	kc := (1 / k) * tau / (tauC + theta)
	tauI := math.Min(tau, 4*(tauC+theta))
	return gains{Kp: kc, Ki: kc / tauI, Kd: 0}
}

// amigoPI is the AMIGO (Åström & Hägglund) PI rule for a FOPDT plant.
//
//	Kc  = (1/K)(0.15 + 0.35 tau/theta - tau^2/(theta+tau)^2)
//	TauI = (0.35 + 13 tau^2/(tau^2 + 12 theta tau + 7 theta^2)) theta
func amigoPI(k, tau, theta float64) gains {
	kc := (1 / k) * (0.15 + 0.35*tau/theta - (tau*tau)/((theta+tau)*(theta+tau)))
	tauI := (0.35 + 13*tau*tau/(tau*tau+12*theta*tau+7*theta*theta)) * theta
	return gains{Kp: kc, Ki: kc / tauI, Kd: 0}
}

// amigoPID is the AMIGO (Åström & Hägglund) PID rule for a FOPDT plant.
//
//	Kc  = (1/K)(0.2 + 0.45 tau/theta)
//	TauI = (0.4 theta + 0.8 tau)/(theta + 0.1 tau) * theta
//	TauD = 0.5 theta tau/(0.3 theta + tau)
func amigoPID(k, tau, theta float64) gains {
	kc := (1 / k) * (0.2 + 0.45*tau/theta)
	tauI := (0.4*theta + 0.8*tau) / (theta + 0.1*tau) * theta
	tauD := 0.5 * theta * tau / (0.3*theta + tau)
	return gains{Kp: kc, Ki: kc / tauI, Kd: kc * tauD}
}
