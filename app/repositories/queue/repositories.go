package queue

import (
	"fmt"

	"github.com/adrianozp/gaardrail/app/entities"
	inmemoryrepo "github.com/adrianozp/gaardrail/app/repositories/queue/inmemory"
	kafkarepo "github.com/adrianozp/gaardrail/app/repositories/queue/kafka"
	kafkaclient "github.com/adrianozp/gaardrail/internal/kafka"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type Queue interface {
	Enqueue(entities.Message) (string, error)
	Dequeue() (entities.Message, error)
	Ack(entities.Message) error
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
	default:
		return nil, fmt.Errorf("queue: unknown protocol %q", cfg.Queue.Protocol)
	}
}
