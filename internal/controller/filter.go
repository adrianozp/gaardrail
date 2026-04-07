package controller

// MovingAverage is a circular-buffer moving average filter.
// It returns the average of the last N values added.
type MovingAverage struct {
	buf   []float64
	size  int
	pos   int
	count int
	sum   float64
}

// NewMovingAverage creates a MovingAverage with the given window size.
// Panics if size < 1.
func NewMovingAverage(size int) *MovingAverage {
	if size < 1 {
		panic("MovingAverage: size must be >= 1")
	}
	return &MovingAverage{
		buf:  make([]float64, size),
		size: size,
	}
}

// Filter adds v to the window and returns the current average.
func (m *MovingAverage) Filter(v float64) float64 {
	// Subtract the value being overwritten from the sum.
	if m.count == m.size {
		m.sum -= m.buf[m.pos]
	}
	m.buf[m.pos] = v
	m.sum += v
	m.pos = (m.pos + 1) % m.size
	if m.count < m.size {
		m.count++
	}
	return m.sum / float64(m.count)
}
