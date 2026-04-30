package queue

import (
	"context"

	"github.com/adrianozp/gaardrail/app/entities"
)

type Queue interface {
	Enqueue(entities.Message) (string, error)
	Dequeue(context.Context) (entities.Message, error)
	Ack(context.Context, entities.Message) error
	Size() (int64, error)
}
