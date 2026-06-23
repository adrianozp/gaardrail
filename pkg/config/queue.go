package config

type Queue struct {
	Protocol string `mapstructure:"protocol" default:"kafka" validate:"required"`
	Capacity int    `mapstructure:"capacity" default:"1000"`
	Query    string `mapstructure:"query"`
}

func init() {
	envKeys = append(envKeys, "queue.protocol", "queue.capacity", "queue.query")
}

// TypeCode maps a queue protocol to a numeric code for dashboard display.
func TypeCode(protocol string) float64 {
	switch protocol {
	case "kafka":
		return 0
	case "inmemory":
		return 1
	case "sqs":
		return 2
	case "constant":
		return 3
	default:
		return -1
	}
}
