package modules

import (
	createMessageHandler "github.com/adrianozp/gaardrail/app/handlers/createmessage"
	floodHandler "github.com/adrianozp/gaardrail/app/handlers/floodmessage"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"go.uber.org/fx"
)

func MessageFactories() fx.Option {
	return fx.Provide(
		createMessageHandler.NewCreateMessageHandler,
		createmessage.NewCreateMessageUseCase,
		floodHandler.NewFloodMessageHandler,
	)
}

func MessageInjections() fx.Option {
	return fx.Provide(
		func(uc createmessage.CreateMessageUseCase) createMessageHandler.CreateMessageUseCase { return uc },
		func(repo queuerepo.Queue) createmessage.Queue { return repo },
		func(uc createmessage.CreateMessageUseCase) floodHandler.CreateMessageUseCase { return uc },
	)
}

func MessageEndpoints() fx.Option {
	return fx.Module("message",
		fx.Invoke(createMessageHandler.RegisterCreateMessageRoutes),
		fx.Invoke(floodHandler.RegisterFloodMessageRoutes),
	)
}
