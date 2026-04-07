package prometheus

import (
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	gauges   sync.Map // map[string]prom.Gauge
	counters sync.Map // map[string]prom.Counter
)

// Recorder is the Prometheus implementation of internal/metrics.Recorder.
// Metrics are registered lazily with the "gaardrail_" prefix.
type Recorder struct{}

func (Recorder) Gauge(values map[string]float64) {
	for name, value := range values {
		getOrRegisterGauge(name).Set(value)
	}
}

func (Recorder) Incr(names []string) {
	for _, name := range names {
		getOrRegisterCounter(name).Inc()
	}
}

func getOrRegisterGauge(name string) prom.Gauge {
	if v, ok := gauges.Load(name); ok {
		return v.(prom.Gauge)
	}
	g := prom.NewGauge(prom.GaugeOpts{
		Name: "gaardrail_" + name,
		Help: name,
	})
	if err := prom.Register(g); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			existing := are.ExistingCollector.(prom.Gauge)
			gauges.Store(name, existing)
			return existing
		}
		panic(err)
	}
	gauges.Store(name, g)
	return g
}

func getOrRegisterCounter(name string) prom.Counter {
	if v, ok := counters.Load(name); ok {
		return v.(prom.Counter)
	}
	c := prom.NewCounter(prom.CounterOpts{
		Name: "gaardrail_" + name,
		Help: name,
	})
	if err := prom.Register(c); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			existing := are.ExistingCollector.(prom.Counter)
			counters.Store(name, existing)
			return existing
		}
		panic(err)
	}
	counters.Store(name, c)
	return c
}
