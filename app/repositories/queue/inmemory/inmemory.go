package inmemory

import (
	"context"
	"sync/atomic"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type Queue struct {
	ch   chan entities.Message
	size atomic.Int64
}

func New(cfg config.Config) *Queue {
	return &Queue{
		ch: make(chan entities.Message, cfg.Queue.Capacity),
	}
}

func (q *Queue) Enqueue(m entities.Message) (string, error) {
	q.ch <- m
	q.size.Add(1)
	return m.ID, nil
}

func (q *Queue) Dequeue(ctx context.Context) (entities.Message, error) {
	select {
	case <-ctx.Done():
		return entities.Message{}, ctx.Err()
	case m := <-q.ch:
		return m, nil
	}
}

func (q *Queue) Ack(_ context.Context, m entities.Message) error {
	q.size.Add(-1)
	return nil
}

func (q *Queue) Size() (int64, error) {
	return q.size.Load(), nil
}
