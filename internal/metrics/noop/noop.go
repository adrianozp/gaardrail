package noop

type Recorder struct{}

func (Recorder) Gauge(map[string]float64) {}
func (Recorder) Incr([]string)            {}
