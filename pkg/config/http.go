package config

type HTTP struct {
	Addr     string `mapstructure:"addr"      default:":8080"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

func (h HTTP) TLSEnabled() bool {
	return h.CertFile != "" && h.KeyFile != ""
}

func init() {
	envKeys = append(envKeys, "http.addr", "http.cert_file", "http.key_file")
}
