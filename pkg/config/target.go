package config

type Target struct {
	Protocol string    `mapstructure:"protocol" default:"http" validate:"required"`
	BaseURL  string    `mapstructure:"base_url" default:"http://localhost:9090"`
	Path     string    `mapstructure:"path"     default:"/events"`
	DSN      string    `mapstructure:"dsn"`
	Driver   string    `mapstructure:"driver"   default:"mysql"`
	TLS      TargetTLS `mapstructure:"tls"`
}

type TargetTLS struct {
	Enabled    bool   `mapstructure:"enabled"`
	CAFile     string `mapstructure:"ca_file"`
	SkipVerify bool   `mapstructure:"skip_verify"`
	ServerName string `mapstructure:"server_name"`
}

func init() {
	envKeys = append(envKeys,
		"target.protocol", "target.base_url", "target.path", "target.dsn", "target.driver",
		"target.tls.enabled", "target.tls.ca_file", "target.tls.skip_verify", "target.tls.server_name",
	)
}
