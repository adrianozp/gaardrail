package modules

import (
	"fmt"

	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	inmemoryrepo "github.com/adrianozp/gaardrail/app/repositories/queue/inmemory"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/queue/kafka"
	sqsrepo "github.com/adrianozp/gaardrail/app/repositories/queue/sqs"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func QueueFactories() fx.Option {
	return fx.Provide(newQueue)
}

func newQueue(cfg config.Config) (queuerepo.Queue, error) {
	switch cfg.Queue.Protocol {
	case "kafka":
		return kafkarepo.New(cfg)
	case "inmemory":
		return inmemoryrepo.New(cfg), nil
	case "sqs":
		return sqsrepo.New(cfg)
	default:
		return nil, fmt.Errorf("queue: unknown protocol %q", cfg.Queue.Protocol)
	}
}
