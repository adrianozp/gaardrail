package modules

import (
	"github.com/adrianozp/gaardrail/app/handlers"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/kafka"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"go.uber.org/fx"
)

func MessageFactories() fx.Option {
	return fx.Provide(
		handlers.NewCreateMessageHandler,
		createmessage.NewCreateMessageUseCase,
	)
}

func MessageInjections() fx.Option {
	return fx.Provide(
		func(uc createmessage.CreateMessageUseCase) handlers.CreateMessageUseCase { return uc },
		func(repo *kafkarepo.KafkaRepository) createmessage.Queue { return repo },
	)
}

func MessageEndpoints() fx.Option {
	return fx.Module("message",
		fx.Invoke(handlers.RegisterCreateMessageRoutes),
	)
}
