package entities

import "time"

type Message struct {
	ID        string
	Ack       func() error
	Body      []byte
	CreatedAt time.Time
}

type Response struct {
	ID        string
	Body      []byte
	CreatedAt time.Time
	PushedAt  time.Time
}

type Metrics struct {
	MeasureTime time.Time
	Metrics     map[string]float64
}

type ControllerParams struct {
	Kp       *float64
	Ki       *float64
	Kd       *float64
	Min      *float64
	Max      *float64
	IClamp   *float64
	Setpoint *float64

	SetpointFilterType *string
	SetpointFilterSize *int

	FfGain *float64

	ModelK        *float64
	ModelTau      *float64
	ModelTheta    *float64
	SampleSeconds *float64
}
