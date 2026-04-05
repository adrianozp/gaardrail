package inmemory

import (
	"sync/atomic"

	"github.com/adrianozp/gaardrail/app/entities"
)

type Queue struct {
	ch   chan entities.Message
	size atomic.Int64
}

func NewQueue(capacity int) *Queue {
	return &Queue{
		ch: make(chan entities.Message, capacity),
	}
}

func (q *Queue) Enqueue(m entities.Message) (string, error) {
	q.ch <- m
	q.size.Add(1)
	return m.ID, nil
}

func (q *Queue) Dequeue() (entities.Message, error) {
	m := <-q.ch
	return m, nil
}

func (q *Queue) Ack(m entities.Message) error {
	q.size.Add(-1)
	return nil
}

func (q *Queue) Size() (int64, error) {
	return q.size.Load(), nil
}
