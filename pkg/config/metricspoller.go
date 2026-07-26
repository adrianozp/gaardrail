package config

type MetricsPoller struct {
	Enabled    bool              `mapstructure:"enabled"     default:"false"`
	IntervalMs int               `mapstructure:"interval_ms" default:"5000"`
	Endpoint   string            `mapstructure:"endpoint"`
	Protocol   string            `mapstructure:"protocol"    default:"exporter"`
	Mappings   map[string]string `mapstructure:"mappings"` // config-file only, not env-overridable
	FilterType string            `mapstructure:"filter_type" default:"none"`
	FilterSize int               `mapstructure:"filter_size" default:"1"`
}

func init() {
	envKeys = append(envKeys, "metrics_poller.enabled", "metrics_poller.interval_ms", "metrics_poller.endpoint", "metrics_poller.protocol", "metrics_poller.filter_type", "metrics_poller.filter_size")
}
