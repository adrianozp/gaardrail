package modules

import (
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/kafka"
	"github.com/adrianozp/gaardrail/pkg/config"
	kafkaclient "github.com/adrianozp/gaardrail/internal/kafka"
	"go.uber.org/fx"
)

func newKafkaClient(cfg config.Config) (*kafkaclient.Client, error) {
	return kafkaclient.New(kafkaclient.Config{
		Brokers:   cfg.Kafka.Brokers,
		Topic:     cfg.Kafka.Topic,
		Partition: cfg.Kafka.Partition,
		GroupID:   cfg.Kafka.GroupID,
	})
}

func KafkaFactories() fx.Option {
	return fx.Provide(
		newKafkaClient,
		kafkarepo.NewKafkaRepository,
	)
}
