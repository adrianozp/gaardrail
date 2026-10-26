package switchable

import (
	"context"
	"fmt"
	"sync"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/repositories/queue/constant"
	"github.com/adrianozp/gaardrail/app/repositories/queue/inmemory"
	"github.com/adrianozp/gaardrail/app/repositories/queue/kafka"
	"github.com/adrianozp/gaardrail/app/repositories/queue/sqs"
	"github.com/adrianozp/gaardrail/pkg/config"
)

var availableTypes = []string{"kafka", "inmemory", "constant", "sqs"}

type Queue struct {
	mu         sync.RWMutex
	active     queue.Queue
	activeType string
	switched   chan struct{}

	buildMu      sync.Mutex
	built        map[string]queue.Queue
	constructors map[string]func() (queue.Queue, error)
}

func New(cfg config.Config, cq *constant.Queue) (*Queue, error) {
	q := &Queue{
		switched: make(chan struct{}),
		built:    map[string]queue.Queue{},
		constructors: map[string]func() (queue.Queue, error){
			"kafka":    func() (queue.Queue, error) { return kafka.New(cfg) },
			"inmemory": func() (queue.Queue, error) { return inmemory.New(cfg), nil },
			"constant": func() (queue.Queue, error) { return cq, nil },
			"sqs":      func() (queue.Queue, error) { return sqs.New(cfg) },
		},
	}

	active, err := q.instance(cfg.Queue.Protocol)
	if err != nil {
		return nil, err
	}
	q.active = active
	q.activeType = cfg.Queue.Protocol
	return q, nil
}

func (q *Queue) instance(t string) (queue.Queue, error) {
	q.buildMu.Lock()
	defer q.buildMu.Unlock()

	if inst, ok := q.built[t]; ok {
		return inst, nil
	}
	ctor, ok := q.constructors[t]
	if !ok {
		return nil, fmt.Errorf("queue: unknown type %q", t)
	}
	inst, err := ctor()
	if err != nil {
		return nil, fmt.Errorf("queue: build %q: %w", t, err)
	}
	q.built[t] = inst
	return inst, nil
}

func (q *Queue) current() queue.Queue {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.active
}

func (q *Queue) Enqueue(m entities.Message) (string, error) {
	return q.current().Enqueue(m)
}

func (q *Queue) Dequeue(ctx context.Context) (entities.Message, error) {
	for {
		active, switched := q.snapshot()
		m, err := dequeueUntilSwitch(ctx, active, switched)
		if err == nil || ctx.Err() != nil || !isClosed(switched) {
			return m, err
		}
	}
}

func (q *Queue) snapshot() (queue.Queue, chan struct{}) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.active, q.switched
}

func dequeueUntilSwitch(ctx context.Context, active queue.Queue, switched chan struct{}) (entities.Message, error) {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-switched:
			cancel()
		case <-childCtx.Done():
		}
	}()
	return active.Dequeue(childCtx)
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func (q *Queue) Ack(ctx context.Context, m entities.Message) error {
	return q.current().Ack(ctx, m)
}

func (q *Queue) Size() (int64, error) {
	return q.current().Size()
}

func (q *Queue) Type() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.activeType
}

func (q *Queue) Available() []string {
	return availableTypes
}

func (q *Queue) SetType(t string) error {
	next, err := q.instance(t)
	if err != nil {
		return err
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if t == q.activeType {
		return nil
	}

	q.active = next
	q.activeType = t
	close(q.switched)
	q.switched = make(chan struct{})
	return nil
}
