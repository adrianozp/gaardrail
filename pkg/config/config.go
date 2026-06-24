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
	Queue         Queue         `mapstructure:"queue"`
	Kafka         Kafka         `mapstructure:"kafka"`
	SQS           SQS           `mapstructure:"sqs"`
	CloudWatch    CloudWatch    `mapstructure:"cloudwatch"`
	Target        Target        `mapstructure:"target"`
	MetricsPoller MetricsPoller `mapstructure:"metrics_poller"`
	Controller    Controller    `mapstructure:"controller"`
	PID           PID           `mapstructure:"pid"`
	Orchestrator  Orchestrator  `mapstructure:"orchestrator"`
	Grafana       Grafana       `mapstructure:"grafana"`

	// Path is the resolved config file used at load time, set after reading. It
	// is the target for runtime persistence and is not bound to any yaml key.
	Path string `mapstructure:"-"`
}

// envKeys is populated by each domain file's init() before Load() is ever called.
var envKeys []string

func Load() (Config, error) {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return Config{}, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")
	viper.AddConfigPath(os.Getenv("APP_PATH"))

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("config: no config file found, using defaults and env vars")
	}

	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	for _, k := range envKeys {
		_ = viper.BindEnv(k)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return Config{}, err
	}

	cfg.Path = viper.ConfigFileUsed()

	return cfg, nil
}
