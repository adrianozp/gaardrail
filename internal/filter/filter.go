// Package filter provides small signal filters used by the controllers to
// smooth the (noisy) process-variable measurement before the control law.
package filter

import (
	"fmt"
	"math"
)

// MovingAverage is a fixed-window moving average over the last N samples.
// A size <= 1 disables filtering: Filter returns its input unchanged.
type MovingAverage struct {
	size int
	buf  []float64
	idx  int
	n    int
	sum  float64
}

// NewMovingAverage creates a moving-average filter of the given window size.
// Sizes < 1 are clamped to 1 (no filtering).
func NewMovingAverage(size int) *MovingAverage {
	if size < 1 {
		size = 1
	}
	return &MovingAverage{size: size, buf: make([]float64, size)}
}

// Size returns the configured window length.
func (m *MovingAverage) Size() int { return m.size }

// Filter pushes x into the window and returns the average of the samples seen
// so far (up to the window size). With size <= 1 it is a passthrough.
func (m *MovingAverage) Filter(x float64) float64 {
	if m.size <= 1 {
		return x
	}
	if m.n == m.size {
		m.sum -= m.buf[m.idx]
	} else {
		m.n++
	}
	m.buf[m.idx] = x
	m.sum += x
	m.idx = (m.idx + 1) % m.size
	return m.sum / float64(m.n)
}

// Reset clears the window.
func (m *MovingAverage) Reset() {
	m.idx = 0
	m.n = 0
	m.sum = 0
	for i := range m.buf {
		m.buf[i] = 0
	}
}

func (m *MovingAverage) FilterFullWindow(x float64) float64 {
	if m.size <= 1 {
		return x
	}
	if m.n == m.size {
		m.sum -= m.buf[m.idx]
	} else {
		m.n++
	}
	m.buf[m.idx] = x
	m.sum += x
	m.idx = (m.idx + 1) % m.size
	return m.sum / float64(m.size)
}

func (m *MovingAverage) fill(v float64) {
	m.n = m.size
	m.sum = v * float64(m.size)
	m.idx = 0
	for i := range m.buf {
		m.buf[i] = v
	}
}

type Signal struct {
	kind          string
	size          int
	seedFromFirst bool
	seeded        bool
	ma            *MovingAverage
	state         float64
}

func NewReferenceFilter(kind string, size int) (*Signal, error) {
	return newSignal(kind, size, false)
}

func NewMeasurementFilter(kind string, size int) (*Signal, error) {
	return newSignal(kind, size, true)
}

func newSignal(kind string, size int, seedFromFirst bool) (*Signal, error) {
	if kind == "" {
		kind = "none"
	}
	if kind != "none" && kind != "moving_average" && kind != "exponential" {
		return nil, fmt.Errorf("filter: tipo desconhecido %q", kind)
	}
	if size < 1 {
		size = 1
	}
	return &Signal{kind: kind, size: size, seedFromFirst: seedFromFirst, ma: NewMovingAverage(size)}, nil
}

func (s *Signal) Filter(x float64) float64 {
	switch s.kind {
	case "moving_average":
		return s.filterMovingAverage(x)
	case "exponential":
		return s.filterExponential(x)
	default:
		return x
	}
}

func (s *Signal) filterMovingAverage(x float64) float64 {
	if s.seedFromFirst && !s.seeded {
		s.Seed(x)
		return x
	}
	return s.ma.FilterFullWindow(x)
}

func (s *Signal) filterExponential(x float64) float64 {
	if s.seedFromFirst && !s.seeded {
		s.Seed(x)
		return x
	}
	a := math.Exp(-1.0 / float64(s.size))
	s.state = a*s.state + (1-a)*x
	s.seeded = true
	return s.state
}

func (s *Signal) Seed(v float64) {
	s.state = v
	s.seeded = true
	s.ma.fill(v)
}

func (s *Signal) Reset()       { s.state = 0; s.seeded = false; s.ma.Reset() }
func (s *Signal) Kind() string { return s.kind }
func (s *Signal) Size() int    { return s.size }
