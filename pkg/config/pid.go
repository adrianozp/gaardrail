package config

type PID struct {
	Kp       float64 `mapstructure:"kp"          default:"1.0"`
	Ki       float64 `mapstructure:"ki"          default:"0.1"`
	Kd       float64 `mapstructure:"kd"          default:"0.01"`
	Min      float64 `mapstructure:"min"         default:"1.0"`
	Max      float64 `mapstructure:"max"         default:"100.0"`
	IClamp   float64 `mapstructure:"i_clamp"     default:"100.0"`
	Setpoint float64 `mapstructure:"setpoint"    default:"70.0"`
	// FfGain is the plant static gain K for feedforward (u_ff = setpoint/K).
	// 0 disables feedforward.
	FfGain float64 `mapstructure:"ff_gain" default:"0"`
	// SetpointFilterTau is the time constant (s) of a 1st-order filter on the
	// setpoint (soft-start / reference prefilter). 0 = instant setpoint (step).
	// Reduces startup overshoot without affecting disturbance rejection.
	SetpointFilterTau  float64 `mapstructure:"setpoint_filter_tau" default:"0"`
	SetpointFilterType string  `mapstructure:"setpoint_filter_type" default:""`
	SetpointFilterSize int     `mapstructure:"setpoint_filter_size" default:"0"`
}

func init() {
	envKeys = append(envKeys, "pid.kp", "pid.ki", "pid.kd", "pid.min", "pid.max", "pid.i_clamp", "pid.setpoint", "pid.ff_gain", "pid.setpoint_filter_tau", "pid.setpoint_filter_type", "pid.setpoint_filter_size")
}
