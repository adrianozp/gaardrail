package metrics

import "github.com/adrianozp/gaardrail/internal/metrics/noop"

// Recorder is the metrics abstraction. Implementations must be safe for
// concurrent use. Gauge and Incr must never block.
type Recorder interface {
	Gauge(values map[string]float64)
	Incr(names []string)
}

// global is the active recorder. It is set once at startup via SetRecorder
// before any business goroutines start, so no mutex is needed.
var global Recorder = noop.Recorder{}

// SetRecorder replaces the global recorder. Must be called before any
// goroutines invoke Gauge or Incr.
func SetRecorder(r Recorder) { global = r }

// Gauge sets one or more named gauge metrics on the global recorder.
func Gauge(values map[string]float64) { global.Gauge(values) }

// Incr increments one or more named counter metrics by 1 on the global recorder.
func Incr(names []string) { global.Incr(names) }
