// Package filter provides small signal filters used by the controllers to
// smooth the (noisy) process-variable measurement before the control law.
package filter

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
