package modules

import (
	kafkaclient "github.com/adrianozp/gaardrail/internal/kafka"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/kafka"
	"go.uber.org/fx"
)

func KafkaFactories() fx.Option {
	return fx.Provide(
		kafkaclient.New,
		kafkarepo.NewKafkaRepository,
	)
}
