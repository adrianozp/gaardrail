package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	gauges   sync.Map // map[string]prometheus.Gauge
	counters sync.Map // map[string]prometheus.Counter
)

// Gauge sets one or more named gauge metrics.
// Metrics are registered on first use with the "gaardrail_" prefix.
func Gauge(values map[string]float64) {
	for name, value := range values {
		getOrRegisterGauge(name).Set(value)
	}
}

// Incr increments one or more named counter metrics by 1.
// Metrics are registered on first use with the "gaardrail_" prefix.
func Incr(names []string) {
	for _, name := range names {
		getOrRegisterCounter(name).Inc()
	}
}

func getOrRegisterGauge(name string) prometheus.Gauge {
	if v, ok := gauges.Load(name); ok {
		return v.(prometheus.Gauge)
	}
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gaardrail_" + name,
		Help: name,
	})
	if err := prometheus.Register(g); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			existing := are.ExistingCollector.(prometheus.Gauge)
			gauges.Store(name, existing)
			return existing
		}
		panic(err)
	}
	gauges.Store(name, g)
	return g
}

func getOrRegisterCounter(name string) prometheus.Counter {
	if v, ok := counters.Load(name); ok {
		return v.(prometheus.Counter)
	}
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gaardrail_" + name,
		Help: name,
	})
	if err := prometheus.Register(c); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			existing := are.ExistingCollector.(prometheus.Counter)
			counters.Store(name, existing)
			return existing
		}
		panic(err)
	}
	counters.Store(name, c)
	return c
}
