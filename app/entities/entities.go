package entities

import "time"

type Message struct {
	ID  string
	Ack func() error
}

type Metrics struct {
	MeasureTime time.Time
	Metrics     map[string]float64
}
