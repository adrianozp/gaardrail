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

type AsyncRecorder struct {
	inner  Recorder
	ch     chan event
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closed bool
}

func NewAsync(r Recorder, bufSize int) (*AsyncRecorder, func()) {
	a := &AsyncRecorder{
		inner: r,
		ch:    make(chan event, bufSize),
	}
	a.wg.Add(1)
	go a.worker()
	return a, a.stop
}

func (a *AsyncRecorder) stop() {
	a.mu.Lock()
	if !a.closed {
		a.closed = true
		close(a.ch)
	}
	a.mu.Unlock()
	a.wg.Wait()
}

func (a *AsyncRecorder) Gauge(values map[string]float64) {
	a.send(event{kind: kindGauge, gauges: values})
}

func (a *AsyncRecorder) Incr(names []string) {
	a.send(event{kind: kindIncr, names: names})
}

func (a *AsyncRecorder) send(e event) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return
	}
	select {
	case a.ch <- e:
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
