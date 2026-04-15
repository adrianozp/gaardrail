package modules

import (
	"github.com/adrianozp/gaardrail/app/orchestrator"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/repositories/targets"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"go.uber.org/fx"
)

func OrchestratorFactories() fx.Option {
	return fx.Provide(
		consumemessage.NewConsumeMessageUseCase,
		orchestrator.NewOrchestrator,
		targets.NewTarget,
	)
}

func OrchestratorInjections() fx.Option {
	return fx.Provide(
		func(repo queuerepo.Queue) consumemessage.SourceQueue { return repo },
		func(p targets.Pusher) consumemessage.Target { return p },
		func(uc consumemessage.ConsumeMessageUseCase) orchestrator.Consumer { return uc },
		func(o *orchestrator.Orchestrator) processmetrics.Orchestrator { return o },
	)
}
