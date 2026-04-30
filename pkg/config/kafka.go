package config

type Kafka struct {
	Brokers   []string `mapstructure:"brokers"    default:"[\"localhost:9092\"]"`
	Topic     string   `mapstructure:"topic"      default:"messages" validate:"required"`
	Partition int32    `mapstructure:"partition"  default:"0"`
	GroupID   string   `mapstructure:"group_id"   default:"gaardrail" validate:"required"`
}

func init() {
	envKeys = append(envKeys, "kafka.brokers", "kafka.topic", "kafka.partition", "kafka.group_id")
}
