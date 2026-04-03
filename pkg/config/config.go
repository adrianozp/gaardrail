package config

import (
	"log"
	"os"
	"strings"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP  HTTP  `mapstructure:"http"`
	Kafka Kafka `mapstructure:"kafka"`
	Target Target `mapstructure:"target"`
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
	BaseURL string `mapstructure:"base_url" default:"http://localhost:9090" validate:"required"`
	Path    string `mapstructure:"path"     default:"/events"              validate:"required"`
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
		log.Printf("config: no config file found, using defaults and env vars")
	}

	viper.SetEnvPrefix("APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	_ = viper.BindEnv("http.addr")
	_ = viper.BindEnv("kafka.brokers")
	_ = viper.BindEnv("kafka.topic")
	_ = viper.BindEnv("kafka.partition")
	_ = viper.BindEnv("kafka.group_id")
	_ = viper.BindEnv("target.base_url")
	_ = viper.BindEnv("target.path")

	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
