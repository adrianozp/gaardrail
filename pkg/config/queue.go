package config

type Queue struct {
	Protocol string `mapstructure:"protocol" default:"kafka" validate:"required"`
	Capacity int    `mapstructure:"capacity" default:"1000"`
}

func init() {
	envKeys = append(envKeys, "queue.protocol", "queue.capacity")
}
