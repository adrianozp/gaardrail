package config

type SQS struct {
	Region            string `mapstructure:"region"             default:"us-east-1"`
	QueueURL          string `mapstructure:"queue_url"`
	MaxMessages       int32  `mapstructure:"max_messages"       default:"1"  validate:"min=1,max=10"`
	WaitTimeSeconds   int32  `mapstructure:"wait_time_seconds"  default:"20"`
	VisibilityTimeout int32  `mapstructure:"visibility_timeout" default:"30"`
}

func init() {
	envKeys = append(envKeys, "sqs.region", "sqs.queue_url", "sqs.max_messages", "sqs.wait_time_seconds", "sqs.visibility_timeout")
}
