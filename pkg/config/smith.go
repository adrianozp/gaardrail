package config

// Smith holds the parameters of the Smith predictor controller: a PI part
// (designed for the delay-free plant) plus an internal FOPDT model used to
// predict the process output and compensate the transport delay.
type Smith struct {
	Kp       float64 `mapstructure:"kp"          default:"1.0"`
	Ki       float64 `mapstructure:"ki"          default:"0.1"`
	Min      float64 `mapstructure:"min"         default:"0.0"`
	Max      float64 `mapstructure:"max"         default:"100.0"`
	IClamp   float64 `mapstructure:"i_clamp"     default:"100.0"`
	Setpoint float64 `mapstructure:"setpoint"    default:"70.0"`

	// Internal FOPDT model: G(s) = ModelK * e^{-ModelTheta s} / (ModelTau s + 1).
	// Estimated from an open-loop identification experiment.
	ModelK     float64 `mapstructure:"model_k"     default:"1.0"`
	ModelTau   float64 `mapstructure:"model_tau"   default:"1.0"`
	ModelTheta float64 `mapstructure:"model_theta" default:"0.0"`

	// SampleSeconds is the nominal sampling period T used to size the dead-time
	// buffer (delay in samples = round(ModelTheta / SampleSeconds)).
	SampleSeconds float64 `mapstructure:"sample_seconds" default:"1.0"`

	// FilterSize is the moving-average window applied to the measurement
	// (<=1 disables filtering).
	FilterSize int `mapstructure:"filter_size" default:"1"`
}

func init() {
	envKeys = append(envKeys,
		"smith.kp", "smith.ki", "smith.min", "smith.max", "smith.i_clamp", "smith.setpoint",
		"smith.model_k", "smith.model_tau", "smith.model_theta", "smith.sample_seconds",
		"smith.filter_size",
	)
}
