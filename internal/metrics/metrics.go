package metrics

import (
	"github.com/adrianozp/gaardrail/internal/metrics/noop"
	"github.com/rs/zerolog/log"
)

type Recorder interface {
	Gauge(values map[string]float64)
	Incr(names []string)
}

var global Recorder = noop.Recorder{}

func SetRecorder(r Recorder) { global = r }

func Gauge(values map[string]float64) {
	log.Debug().Fields(map[string]any{"values": values}).Msg("metrics: gauge")
	global.Gauge(values)
}

func Incr(names []string) {
	global.Incr(names)
}
