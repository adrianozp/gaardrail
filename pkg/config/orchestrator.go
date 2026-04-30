package config

type Orchestrator struct {
	Rate    int `mapstructure:"rate"    default:"0"`
	Burst   int `mapstructure:"burst"   default:"10"`
	Workers int `mapstructure:"workers" default:"1"`
}

func init() {
	envKeys = append(envKeys, "orchestrator.rate", "orchestrator.burst", "orchestrator.workers")
}
