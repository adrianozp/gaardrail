package noop

// Recorder is a no-op metrics recorder.
// It satisfies internal/metrics.Recorder without importing it (Go duck typing).
type Recorder struct{}

func (Recorder) Gauge(map[string]float64) {}
func (Recorder) Incr([]string)            {}
