package modules

import (
	"context"
	"fmt"

	queuequeryhandler "github.com/adrianozp/gaardrail/app/handlers/queuequery"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	constantrepo "github.com/adrianozp/gaardrail/app/repositories/queue/constant"
	inmemoryrepo "github.com/adrianozp/gaardrail/app/repositories/queue/inmemory"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/queue/kafka"
	sqsrepo "github.com/adrianozp/gaardrail/app/repositories/queue/sqs"
	"github.com/adrianozp/gaardrail/app/usecases/queuequery"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func QueueFactories() fx.Option {
	return fx.Provide(
		constantrepo.New,
		newQueue,
		queuequery.New,
		queuequeryhandler.New,
	)
}

func newQueue(cfg config.Config, cq *constantrepo.Queue) (queuerepo.Queue, error) {
	switch cfg.Queue.Protocol {
	case "kafka":
		return kafkarepo.New(cfg)
	case "inmemory":
		return inmemoryrepo.New(cfg), nil
	case "sqs":
		return sqsrepo.New(cfg)
	case "constant":
		return cq, nil
	default:
		return nil, fmt.Errorf("queue: unknown protocol %q", cfg.Queue.Protocol)
	}
}

func QueueInjections() fx.Option {
	return fx.Provide(
		func(cq *constantrepo.Queue) queuequery.QueryHolder { return cq },
		func(uc queuequery.UseCase) queuequeryhandler.QueueQueryUseCase { return uc },
	)
}

func QueueEndpoints() fx.Option {
	return fx.Module("queue",
		fx.Invoke(queuequeryhandler.RegisterRoutes),
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
