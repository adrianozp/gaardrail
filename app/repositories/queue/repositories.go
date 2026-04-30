package queue

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/entities"
	inmemoryrepo "github.com/adrianozp/gaardrail/app/repositories/queue/inmemory"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/queue/kafka"
	sqsrepo "github.com/adrianozp/gaardrail/app/repositories/queue/sqs"
	kafkaclient "github.com/adrianozp/gaardrail/internal/kafka"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type Queue interface {
	Enqueue(entities.Message) (string, error)
	Dequeue(context.Context) (entities.Message, error)
	Ack(context.Context, entities.Message) error
	Size() (int64, error)
}

func New(cfg config.Config) (Queue, error) {
	switch cfg.Queue.Protocol {
	case "kafka":
		client, err := kafkaclient.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("queue: kafka: %w", err)
		}
		return kafkarepo.NewKafkaRepository(client), nil
	case "inmemory":
		return inmemoryrepo.NewQueue(cfg.Queue.Capacity), nil
	case "sqs":
		repo, err := sqsrepo.New(cfg.SQS)
		if err != nil {
			return nil, fmt.Errorf("queue: sqs: %w", err)
		}
		return repo, nil
	default:
		return nil, fmt.Errorf("queue: unknown protocol %q", cfg.Queue.Protocol)
	}
}
