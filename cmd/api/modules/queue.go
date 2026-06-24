package modules

import (
	"context"

	queuequeryhandler "github.com/adrianozp/gaardrail/app/handlers/queuequery"
	switchqueuehandler "github.com/adrianozp/gaardrail/app/handlers/switchqueue"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	constantrepo "github.com/adrianozp/gaardrail/app/repositories/queue/constant"
	switchablerepo "github.com/adrianozp/gaardrail/app/repositories/queue/switchable"
	"github.com/adrianozp/gaardrail/app/usecases/queuequery"
	"github.com/adrianozp/gaardrail/app/usecases/switchqueue"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func QueueFactories() fx.Option {
	return fx.Provide(
		constantrepo.New,
		switchablerepo.New,
		queuequery.New,
		queuequeryhandler.New,
		switchqueue.New,
		switchqueuehandler.New,
	)
}

func QueueInjections() fx.Option {
	return fx.Provide(
		func(s *switchablerepo.Queue) queuerepo.Queue { return s },
		func(s *switchablerepo.Queue) switchqueue.Queue { return s },
		func(cq *constantrepo.Queue) queuequery.QueryHolder { return cq },
		func(p *config.Persister) queuequery.ConfigStore { return p },
		func(p *config.Persister) switchqueue.ConfigStore { return p },
		func(uc queuequery.UseCase) queuequeryhandler.QueueQueryUseCase { return uc },
		func(uc switchqueue.UseCase) switchqueuehandler.SwitchQueueUseCase { return uc },
	)
}

func QueueEndpoints() fx.Option {
	return fx.Module("queue",
		fx.Invoke(queuequeryhandler.RegisterRoutes),
		fx.Invoke(switchqueuehandler.RegisterRoutes),
	)
}

func QueueLifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, cfg config.Config) {
		lc.Append(fx.Hook{
			OnStart: func(_ context.Context) error {
				metrics.Gauge(map[string]float64{"queue_type": config.TypeCode(cfg.Queue.Protocol)})
				return nil
			},
		})
	})
}
