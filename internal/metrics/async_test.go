package metrics_test

import (
	"sync"
	"testing"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
)

type recordingRecorder struct {
	mu     sync.Mutex
	gauges []map[string]float64
	incrs  [][]string
}

func (r *recordingRecorder) Gauge(values map[string]float64) {
	r.mu.Lock()
	r.gauges = append(r.gauges, values)
	r.mu.Unlock()
}

func (r *recordingRecorder) Incr(names []string) {
	r.mu.Lock()
	r.incrs = append(r.incrs, names)
	r.mu.Unlock()
}

func TestAsyncRecorder_DeliverGauge(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 16)

	async.Gauge(map[string]float64{"cpu": 42.0})
	stop()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.gauges) != 1 {
		t.Fatalf("expected 1 gauge, got %d", len(rec.gauges))
	}
	if rec.gauges[0]["cpu"] != 42.0 {
		t.Errorf("expected cpu=42.0, got %v", rec.gauges[0])
	}
}

func TestAsyncRecorder_DeliverIncr(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 16)

	async.Incr([]string{"messages_total"})
	stop()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.incrs) != 1 {
		t.Fatalf("expected 1 incr, got %d", len(rec.incrs))
	}
	if rec.incrs[0][0] != "messages_total" {
		t.Errorf("expected messages_total, got %v", rec.incrs[0])
	}
}

func TestAsyncRecorder_StopDrainsRemaining(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 64)

	for i := 0; i < 10; i++ {
		async.Gauge(map[string]float64{"tick": float64(i)})
	}
	stop()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.gauges) != 10 {
		t.Errorf("expected 10 gauges after stop, got %d", len(rec.gauges))
	}
}

func TestAsyncRecorder_FullChannelDropsSilently(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 1)

	done := make(chan struct{})
	go func() {
		async.Gauge(map[string]float64{"a": 1})
		async.Gauge(map[string]float64{"b": 2})
		close(done)
	}()
	<-done
	stop()
}

func TestAsyncRecorder_SendAfterStopIsDropped(t *testing.T) {
	rec := &recordingRecorder{}
	async, stop := metrics.NewAsync(rec, 16)
	stop()

	async.Gauge(map[string]float64{"cpu": 1})
	async.Incr([]string{"messages_total"})

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.gauges) != 0 || len(rec.incrs) != 0 {
		t.Fatalf("events recorded after stop: %v %v", rec.gauges, rec.incrs)
	}
}

func TestAsyncRecorder_StopIsIdempotent(t *testing.T) {
	_, stop := metrics.NewAsync(&recordingRecorder{}, 1)
	stop()
	stop()
}

func TestAsyncRecorder_ConcurrentSendAndStop(t *testing.T) {
	async, stop := metrics.NewAsync(&recordingRecorder{}, 1)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { async.Gauge(map[string]float64{"x": 1}) })
	}
	wg.Go(stop)
	wg.Wait()
}
