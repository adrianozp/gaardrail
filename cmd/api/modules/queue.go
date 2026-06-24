package modules

import (
	"context"
	"fmt"

	queuequeryhandler "github.com/adrianozp/gaardrail/app/handlers/queuequery"
	switchqueuehandler "github.com/adrianozp/gaardrail/app/handlers/switchqueue"
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	constantrepo "github.com/adrianozp/gaardrail/app/repositories/queue/constant"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/queue/kafka"
	sqsrepo "github.com/adrianozp/gaardrail/app/repositories/queue/sqs"
	staticqueuerepo "github.com/adrianozp/gaardrail/app/repositories/queue/staticqueue"
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
		newQueue,
		newSwitchQueueSource,
		queuequery.New,
		queuequeryhandler.New,
		switchqueue.New,
		switchqueuehandler.New,
	)
}

func newQueue(cfg config.Config, cq *constantrepo.Queue) (queuerepo.Queue, error) {
	switch cfg.Queue.Protocol {
	case "kafka":
		return kafkarepo.New(cfg)
	case "inmemory", "constant":
		return switchablerepo.New(cfg, cq)
	case "sqs":
		return sqsrepo.New(cfg)
	default:
		return nil, fmt.Errorf("queue: unknown protocol %q", cfg.Queue.Protocol)
	}
}

// newSwitchQueueSource exposes the runtime queue as a switch source. A
// switchable queue (inmemory/constant) is used directly; any other protocol
// (kafka/sqs) falls back to a static, non-switchable source.
func newSwitchQueueSource(q queuerepo.Queue, cfg config.Config) switchqueue.Queue {
	if s, ok := q.(switchqueue.Queue); ok {
		return s
	}
	return staticqueuerepo.New(cfg.Queue.Protocol)
}

func QueueInjections() fx.Option {
	return fx.Provide(
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
