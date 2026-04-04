package entities

import "time"

type Message struct {
	ID        string
	Ack       func() error
	Body      []byte
	CreatedAt time.Time
}

type Metrics struct {
	MeasureTime time.Time
	Metrics     map[string]float64
}
