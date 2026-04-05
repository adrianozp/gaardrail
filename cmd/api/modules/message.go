package modules

import (
	createMessageHandler "github.com/adrianozp/gaardrail/app/handlers/createmessage"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"go.uber.org/fx"
)

func MessageFactories() fx.Option {
	return fx.Provide(
		createMessageHandler.NewCreateMessageHandler,
		createmessage.NewCreateMessageUseCase,
	)
}

func MessageInjections() fx.Option {
	return fx.Provide(
		func(uc createmessage.CreateMessageUseCase) createMessageHandler.CreateMessageUseCase { return uc },
		func(repo queuerepo.Queue) createmessage.Queue { return repo },
	)
}

func MessageEndpoints() fx.Option {
	return fx.Module("message",
		fx.Invoke(createMessageHandler.RegisterCreateMessageRoutes),
	)
}
