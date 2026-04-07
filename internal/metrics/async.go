package metrics

import "sync"

type eventKind uint8

const (
	kindGauge eventKind = iota
	kindIncr
)

type event struct {
	kind   eventKind
	gauges map[string]float64
	names  []string
}

// AsyncRecorder wraps a Recorder and dispatches all sends through a buffered
// channel. Gauge and Incr never block: if the channel is full, the event is
// dropped silently (metrics are best-effort).
type AsyncRecorder struct {
	inner Recorder
	ch    chan event
	wg    sync.WaitGroup
}

// NewAsync returns an AsyncRecorder and a stop function.
// bufSize is the number of events the channel can buffer before drops occur.
// The stop function closes the channel and waits for the worker to drain all
// buffered events before returning. Call it during graceful shutdown.
func NewAsync(r Recorder, bufSize int) (*AsyncRecorder, func()) {
	a := &AsyncRecorder{
		inner: r,
		ch:    make(chan event, bufSize),
	}
	a.wg.Add(1)
	go a.worker()
	stop := func() {
		close(a.ch)
		a.wg.Wait()
	}
	return a, stop
}

func (a *AsyncRecorder) Gauge(values map[string]float64) {
	select {
	case a.ch <- event{kind: kindGauge, gauges: values}:
	default:
	}
}

func (a *AsyncRecorder) Incr(names []string) {
	select {
	case a.ch <- event{kind: kindIncr, names: names}:
	default:
	}
}

func (a *AsyncRecorder) worker() {
	defer a.wg.Done()
	for e := range a.ch {
		switch e.kind {
		case kindGauge:
			a.inner.Gauge(e.gauges)
		case kindIncr:
			a.inner.Incr(e.names)
		}
	}
}
