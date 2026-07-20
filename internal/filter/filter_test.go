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
	// Partial window: average of what's seen so far.
	if got := m.Filter(3); got != 3 {
		t.Errorf("got %v, want 3", got)
	}
	if got := m.Filter(6); got != 4.5 {
		t.Errorf("got %v, want 4.5", got)
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
	if got := m.Filter(4); got != 4 {
		t.Errorf("after reset expected 4, got %v", got)
	}
}

func TestSmoothsNoise(t *testing.T) {
	// A large window should attenuate an alternating signal toward its mean.
	m := NewMovingAverage(10)
	var last float64
	for i := 0; i < 100; i++ {
		x := 50.0
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
