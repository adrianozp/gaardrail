package constant

import (
	"context"
	"sync"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/pkg/uuid"
)

// Queue is a queue connector whose Dequeue always returns a fresh message
// carrying the same configured query. It never drains: the payload is fixed and
// set at startup (config) or at runtime via SetQuery.
type Queue struct {
	mu    sync.RWMutex
	query string
	ready chan struct{}
	once  sync.Once
}

func New(cfg config.Config) *Queue {
	q := &Queue{ready: make(chan struct{})}
	if cfg.Queue.Query != "" {
		q.SetQuery(cfg.Queue.Query)
	}
	return q
}

func (q *Queue) SetQuery(query string) {
	q.mu.Lock()
	q.query = query
	q.mu.Unlock()
	if query != "" {
		q.once.Do(func() { close(q.ready) })
	}
}

func (q *Queue) Query() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.query
}

func (q *Queue) Enqueue(m entities.Message) (string, error) {
	return m.ID, nil
}

func (q *Queue) Dequeue(ctx context.Context) (entities.Message, error) {
	query := q.Query()
	if query == "" {
		select {
		case <-ctx.Done():
			return entities.Message{}, ctx.Err()
		case <-q.ready:
		}
		query = q.Query()
	}

	return entities.Message{
		ID:        uuid.New(),
		Body:      []byte(query),
		CreatedAt: clock.Now(),
	}, nil
}

func (q *Queue) Ack(_ context.Context, _ entities.Message) error {
	return nil
}

// Size returns -1 to signal an infinite queue (not applicable), distinguishing
// it from a genuinely empty queue (0).
func (q *Queue) Size() (int64, error) {
	return -1, nil
}
