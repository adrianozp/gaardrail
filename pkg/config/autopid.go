package config

// AutoPID holds the parameters of the self-tuning (auto-PID) controller. It runs
// an open-loop step experiment on activation, identifies a FOPDT model of the
// plant (two-point Smith method) and computes the PID gains automatically via the
// selected tuning rule, then closes the loop with an embedded PID controller.
//
// Operating limits for the control phase (min/max/i_clamp/setpoint/filter_size)
// are inherited from the pid.* config, so they are not duplicated here.
type AutoPID struct {
	// TuningRule selects the auto-tuning formula: "amigo" (Åström-Hägglund) or
	// "simc" (Skogestad).
	TuningRule string `mapstructure:"tuning_rule" default:"amigo" validate:"oneof=amigo simc"`

	// Mode selects the controller structure produced by the tuning rule: "pi" or
	// "pid". The system runs as PI in production, so "pi" is the default.
	Mode string `mapstructure:"mode" default:"pi" validate:"oneof=pi pid"`

	// TauC is the desired closed-loop time constant (s) for SIMC (tuning
	// aggressiveness). 0 falls back to the effective dead time (theta_eff), the
	// robust Skogestad default. Unused by AMIGO (fixed rule).
	TauC float64 `mapstructure:"tau_c" default:"0"`

	// BaselineOutput is the drain rate held before the step, to establish the
	// output baseline y0. 0 falls back to the plant Min (pid.min).
	BaselineOutput float64 `mapstructure:"baseline_output" default:"0"`

	// StepOutput is the drain rate applied during the step experiment. 0 falls
	// back to the plant Max (pid.max).
	StepOutput float64 `mapstructure:"step_output" default:"0"`

	// BaselineSeconds is how long the baseline is held before the step.
	BaselineSeconds float64 `mapstructure:"baseline_seconds" default:"120" validate:"gt=0"`

	// IdentifySeconds is how long the step is held while the reaction curve is
	// recorded (must cover the settling; ~10 samples at T=60s).
	IdentifySeconds float64 `mapstructure:"identify_seconds" default:"600" validate:"gt=0"`
}

func init() {
	envKeys = append(envKeys,
		"autopid.tuning_rule", "autopid.mode", "autopid.tau_c",
		"autopid.baseline_output", "autopid.step_output",
		"autopid.baseline_seconds", "autopid.identify_seconds",
	)
}
