package config

type Target struct {
	Protocol string `mapstructure:"protocol" default:"http" validate:"required"`
	BaseURL  string `mapstructure:"base_url" default:"http://localhost:9090"`
	Path     string `mapstructure:"path"     default:"/events"`
	DSN      string `mapstructure:"dsn"`
	Driver   string `mapstructure:"driver"   default:"mysql"`
}

func init() {
	envKeys = append(envKeys, "target.protocol", "target.base_url", "target.path", "target.dsn", "target.driver")
}
