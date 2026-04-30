package config

type Grafana struct {
	URL string `mapstructure:"url" default:"http://localhost:3000/d/flood-test/flood-test?kiosk"`
}

func init() {
	envKeys = append(envKeys, "grafana.url")
}
