package modules

import (
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/kafka"
	"github.com/adrianozp/gaardrail/app/repositories/targets"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage"
	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/pkg/config"
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
		func(cfg config.Config) (consumemessage.Target, error) {
			return targets.NewTarget(cfg)
		},
		func(uc consumemessage.ConsumeMessageUseCase) orchestrator.Consumer { return uc },
		func(o *orchestrator.Orchestrator) processmetrics.Orchestrator { return o },
	)
}
