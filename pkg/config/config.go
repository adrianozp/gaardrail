package config

import (
	"os"
	"strings"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP          HTTP          `mapstructure:"http"`
	Kafka         Kafka         `mapstructure:"kafka"`
	Target        Target        `mapstructure:"target"`
	MetricsPoller MetricsPoller `mapstructure:"metrics_poller"`
	PID           PID           `mapstructure:"pid"`
}

type HTTP struct {
	Addr string `mapstructure:"addr" default:":8080"`
}

type Kafka struct {
	Brokers   []string `mapstructure:"brokers" default:"[\"localhost:9092\"]"`
	Topic     string   `mapstructure:"topic"     default:"messages"  validate:"required"`
	Partition int32    `mapstructure:"partition" default:"0"`
	GroupID   string   `mapstructure:"group_id"  default:"gaardrail" validate:"required"`
}

type Target struct {
	Protocol string `mapstructure:"protocol" default:"http" validate:"required"`
	BaseURL  string `mapstructure:"base_url" default:"http://localhost:9090"`
	Path     string `mapstructure:"path"     default:"/events"`
	DSN      string `mapstructure:"dsn"`
}

type MetricsPoller struct {
	Enabled    bool              `mapstructure:"enabled"     default:"false"`
	IntervalMs int               `mapstructure:"interval_ms" default:"5000"`
	Endpoint   string            `mapstructure:"endpoint"`
	Protocol   string            `mapstructure:"protocol"    default:"prometheus"`
	Mappings   map[string]string `mapstructure:"mappings"`
}

type PID struct {
	Kp       float64 `mapstructure:"kp"        default:"1.0"`
	Ki       float64 `mapstructure:"ki"        default:"0.1"`
	Kd       float64 `mapstructure:"kd"        default:"0.01"`
	Min      float64 `mapstructure:"min"       default:"1.0"`
	Max      float64 `mapstructure:"max"       default:"100.0"`
	IClamp   float64 `mapstructure:"i_clamp"   default:"100.0"`
	Setpoint float64 `mapstructure:"setpoint"  default:"70.0"`
}

func Load() (Config, error) {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return Config{}, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(os.Getenv("APP_PATH"))

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("config: no config file found, using defaults and env vars")
	}

	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	_ = viper.BindEnv("http.addr")
	_ = viper.BindEnv("kafka.brokers")
	_ = viper.BindEnv("kafka.topic")
	_ = viper.BindEnv("kafka.partition")
	_ = viper.BindEnv("kafka.group_id")
	_ = viper.BindEnv("target.protocol")
	_ = viper.BindEnv("target.base_url")
	_ = viper.BindEnv("target.path")
	_ = viper.BindEnv("target.dsn")
	_ = viper.BindEnv("metrics_poller.enabled")
	_ = viper.BindEnv("metrics_poller.interval_ms")
	_ = viper.BindEnv("metrics_poller.endpoint")
	_ = viper.BindEnv("metrics_poller.protocol")
	_ = viper.BindEnv("pid.kp")
	_ = viper.BindEnv("pid.ki")
	_ = viper.BindEnv("pid.kd")
	_ = viper.BindEnv("pid.min")
	_ = viper.BindEnv("pid.max")
	_ = viper.BindEnv("pid.i_clamp")
	_ = viper.BindEnv("pid.setpoint")

	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
