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

	// FilterSize is the moving-average window on the measurement (all controllers).
	FilterSize *int

	// FfGain is the plant static gain K used for feedforward: u_ff = setpoint/K.
	// Zero disables feedforward. (PID controller.)
	FfGain *float64

	// Smith predictor internal FOPDT model (nil for other controllers).
	ModelK        *float64
	ModelTau      *float64
	ModelTheta    *float64
	SampleSeconds *float64
}
