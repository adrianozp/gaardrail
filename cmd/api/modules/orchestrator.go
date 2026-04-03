package modules

import (
	httprepo "github.com/adrianozp/gaardrail/app/repositories/http"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/kafka"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage"
	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"go.uber.org/fx"
)

func OrchestratorFactories() fx.Option {
	return fx.Provide(
		consumemessage.NewConsumeMessageUseCase,
		orchestrator.NewOrchestrator,
	)
}

func OrchestratorInjections() fx.Option {
	return fx.Provide(
		func(repo *kafkarepo.KafkaRepository) consumemessage.Queue { return repo },
		func(repo *httprepo.HTTPRepository) consumemessage.Target { return repo },
		func(uc consumemessage.ConsumeMessageUseCase) orchestrator.Consumer { return uc },
		func(o *orchestrator.Orchestrator) processmetrics.Orchestrator { return o },
	)
}
