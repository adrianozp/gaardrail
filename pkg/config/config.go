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
	Smith         Smith         `mapstructure:"smith"`
	Orchestrator  Orchestrator  `mapstructure:"orchestrator"`

	Path string `mapstructure:"-"`
}

var envKeys []string

func Load() (Config, error) {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return Config{}, err
	}
	readConfigFile()
	bindEnv()
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	if err := validator.New().Struct(cfg); err != nil {
		return Config{}, err
	}
	cfg.Path = viper.ConfigFileUsed()
	return cfg, nil
}

func readConfigFile() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")
	viper.AddConfigPath(os.Getenv("APP_PATH"))
	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("config: no config file found, using defaults and env vars")
	}
}

func bindEnv() {
	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	for _, k := range envKeys {
		_ = viper.BindEnv(k)
	}
}
