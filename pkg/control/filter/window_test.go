package filter

import (
	"math"
	"testing"
)

func assertPassthrough(t *testing.T, size int) {
	t.Helper()
	w := newWindow(size)
	for _, x := range []float64{5, 10, -2} {
		if got := w.push(x); got != x {
			t.Errorf("size %d: expected passthrough %v, got %v", size, x, got)
		}
	}
}

func TestWindowPassthroughWhenSizeLEOne(t *testing.T) {
	for _, size := range []int{0, 1, -3} {
		assertPassthrough(t, size)
	}
}

func TestWindowAveragesOverFullWindow(t *testing.T) {
	w := newWindow(3)
	inputs := []float64{3, 6, 9, 12}
	want := []float64{1, 3, 6, 9}
	for i, x := range inputs {
		if got := w.push(x); got != want[i] {
			t.Errorf("sample %d: got %v, want %v", i, got, want[i])
		}
	}
}

func TestWindowReset(t *testing.T) {
	w := newWindow(3)
	w.push(10)
	w.push(20)
	w.reset()
	if got, want := w.push(4), 4.0/3.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("after reset expected %v, got %v", want, got)
	}
}

func TestWindowSmoothsAlternatingSignal(t *testing.T) {
	w := newWindow(10)
	var last float64
	for i := range 100 {
		last = w.push(float64(i%2) * 100)
	}
	if math.Abs(last-50) > 10 {
		t.Errorf("expected filtered value near 50, got %v", last)
	}
}
