package filter

import (
	"math"
	"testing"
)

func TestPassthroughWhenSizeLEOne(t *testing.T) {
	for _, sz := range []int{0, 1, -3} {
		m := NewMovingAverage(sz)
		for _, x := range []float64{5, 10, -2} {
			if got := m.Filter(x); got != x {
				t.Errorf("size %d: expected passthrough %v, got %v", sz, x, got)
			}
		}
	}
}

func TestMovingAverage(t *testing.T) {
	m := NewMovingAverage(3)
	// The average is always over the full window; unfilled slots count as zero.
	if got := m.Filter(3); got != 1 { // (3+0+0)/3
		t.Errorf("got %v, want 1", got)
	}
	if got := m.Filter(6); got != 3 { // (3+6+0)/3
		t.Errorf("got %v, want 3", got)
	}
	if got := m.Filter(9); got != 6 { // (3+6+9)/3
		t.Errorf("got %v, want 6", got)
	}
	// Window full: oldest (3) drops out. (6+9+12)/3 = 9
	if got := m.Filter(12); got != 9 {
		t.Errorf("got %v, want 9", got)
	}
}

func TestReset(t *testing.T) {
	m := NewMovingAverage(3)
	m.Filter(10)
	m.Filter(20)
	m.Reset()
	if got, want := m.Filter(4), 4.0/3.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("after reset expected %v, got %v", want, got)
	}
}

func TestSmoothsNoise(t *testing.T) {
	// A large window should attenuate an alternating signal toward its mean.
	m := NewMovingAverage(10)
	var last float64
	for i := range 100 {
		var x float64
		if i%2 == 0 {
			x = 0
		} else {
			x = 100
		}
		last = m.Filter(x)
	}
	if math.Abs(last-50) > 10 {
		t.Errorf("expected filtered value near 50, got %v", last)
	}
}

func TestSignalNonePassthrough(t *testing.T) {
	s, err := NewReferenceFilter("none", 5)
	if err != nil || s.Filter(50) != 50 {
		t.Fatalf("none must be a passthrough, err=%v", err)
	}
}

func TestSignalMovingAverageRampsStepOverNSamples(t *testing.T) {
	s, _ := NewReferenceFilter("moving_average", 4)
	got := []float64{s.Filter(50), s.Filter(50), s.Filter(50), s.Filter(50)}
	want := []float64{12.5, 25, 37.5, 50}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("sample %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestSignalExponentialConstantPerTick(t *testing.T) {
	s, _ := NewReferenceFilter("exponential", 2)
	a := math.Exp(-0.5)
	if got, want := s.Filter(50), (1-a)*50; math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSignalMeasurementSeedsWithFirstSample(t *testing.T) {
	s, _ := NewMeasurementFilter("exponential", 4)
	if got := s.Filter(80); got != 80 {
		t.Fatalf("first sample must pass through, got %v", got)
	}
}

func TestSignalInvalidKind(t *testing.T) {
	if _, err := NewReferenceFilter("median", 3); err == nil {
		t.Fatal("invalid kind must return an error")
	}
}

func TestSignalSeedSetsState(t *testing.T) {
	s, _ := NewReferenceFilter("exponential", 3)
	s.Seed(50)
	if got := s.Filter(50); math.Abs(got-50) > 1e-9 {
		t.Fatalf("after Seed(50), Filter(50) must hold 50, got %v", got)
	}
}
