package metricstest

import (
	"sync"
	"testing"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/internal/metrics/noop"
)

type Recorder struct {
	mu     sync.Mutex
	gauges map[string]float64
}

func Capture(t *testing.T) *Recorder {
	t.Helper()
	r := &Recorder{gauges: map[string]float64{}}
	metrics.SetRecorder(r)
	t.Cleanup(func() { metrics.SetRecorder(noop.Recorder{}) })
	return r
}

func (r *Recorder) Gauge(values map[string]float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range values {
		r.gauges[k] = v
	}
}

func (r *Recorder) Incr([]string) {}

func (r *Recorder) Value(name string) (float64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.gauges[name]
	return v, ok
}
