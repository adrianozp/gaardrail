package config

type Target struct {
	Protocol string    `mapstructure:"protocol" default:"http" validate:"required"`
	BaseURL  string    `mapstructure:"base_url" default:"http://localhost:9090"`
	Path     string    `mapstructure:"path"     default:"/events"`
	DSN      string    `mapstructure:"dsn"`
	Driver   string    `mapstructure:"driver"   default:"mysql"`
	TLS      TargetTLS `mapstructure:"tls"`
}

// TargetTLS configures encrypted communication with the SQL target.
// Only the mysql driver is supported. When Enabled is false the connection
// stays a plain TCP connection (no behaviour change).
type TargetTLS struct {
	Enabled    bool   `mapstructure:"enabled"`
	CAFile     string `mapstructure:"ca_file"`     // optional PEM CA bundle for server verification
	SkipVerify bool   `mapstructure:"skip_verify"` // encrypt but do not verify the server certificate (testing)
	ServerName string `mapstructure:"server_name"` // optional, used for certificate verification
}

func init() {
	envKeys = append(envKeys,
		"target.protocol", "target.base_url", "target.path", "target.dsn", "target.driver",
		"target.tls.enabled", "target.tls.ca_file", "target.tls.skip_verify", "target.tls.server_name",
	)
}
