package config

type CloudWatch struct {
	Region     string            `mapstructure:"region"     default:"us-east-1"`
	Period     int32             `mapstructure:"period"     default:"60"`
	Dimensions map[string]string `mapstructure:"dimensions"` // config-file only, not env-overridable
}

func init() {
	envKeys = append(envKeys, "cloudwatch.region", "cloudwatch.period")
}
