package config

type PID struct {
	Kp         float64 `mapstructure:"kp"          default:"1.0"`
	Ki         float64 `mapstructure:"ki"          default:"0.1"`
	Kd         float64 `mapstructure:"kd"          default:"0.01"`
	Min        float64 `mapstructure:"min"         default:"1.0"`
	Max        float64 `mapstructure:"max"         default:"100.0"`
	IClamp     float64 `mapstructure:"i_clamp"     default:"100.0"`
	Setpoint   float64 `mapstructure:"setpoint"    default:"70.0"`
	FilterSize int     `mapstructure:"filter_size" default:"1"`
}

func init() {
	envKeys = append(envKeys, "pid.kp", "pid.ki", "pid.kd", "pid.min", "pid.max", "pid.i_clamp", "pid.setpoint", "pid.filter_size")
}
