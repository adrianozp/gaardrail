package config

type Controller struct {
	Type string `mapstructure:"type" default:"pid"`
}

func init() {
	envKeys = append(envKeys, "controller.type")
}
