// Package switchable wraps the queues that can be selected at runtime and lets
// the active one be swapped without restarting. It implements queue.Queue by
// delegating every call to the currently selected queue, so consumers never
// know the underlying implementation changed. The switch logic and all edge
// cases live here, keeping use cases and FX modules free of branching.
//
// Queues are built lazily: only the queue configured at boot is constructed up
// front; the others (which may need external connections, e.g. kafka/sqs) are
// constructed and cached on the first switch to them. A construction failure
// surfaces as an error and leaves the active queue unchanged.
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

// availableTypes is the fixed, ordered set of runtime-switchable queues.
var availableTypes = []string{"kafka", "inmemory", "constant", "sqs"}

// Queue delegates to whichever underlying queue is currently active. Each
// underlying instance is built at most once and stays alive across switches,
// keeping its own state.
type Queue struct {
	mu         sync.RWMutex
	active     queue.Queue
	activeType string
	// switched is closed on every successful SetType to release a Dequeue parked
	// on the previous (possibly blocking) queue; it is then replaced.
	switched chan struct{}

	// buildMu guards lazy construction so a slow connect (kafka/sqs) does not
	// block delegation on mu.
	buildMu      sync.Mutex
	built        map[string]queue.Queue
	constructors map[string]func() (queue.Queue, error)
}

// New wires the constructors for every switchable queue and eagerly builds the
// one named by cfg.Queue.Protocol. The constant queue is passed in so it can be
// shared with the queue-query holder. Returns an error if the configured queue
// is unknown or fails to build.
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

// instance returns the lazily built, cached queue for a type.
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

// Dequeue delegates to the active queue but stays responsive to switches: a
// switch cancels the in-flight Dequeue (via switched) so a worker blocked on an
// empty/idle queue is released and retries against the newly active queue. A
// cancellation of the caller's ctx is propagated as-is.
func (q *Queue) Dequeue(ctx context.Context) (entities.Message, error) {
	for {
		q.mu.RLock()
		active := q.active
		switched := q.switched
		q.mu.RUnlock()

		childCtx, cancel := context.WithCancel(ctx)
		go func() {
			select {
			case <-switched:
				cancel()
			case <-childCtx.Done():
			}
		}()

		m, err := active.Dequeue(childCtx)
		cancel()

		// Retry only when the child context was cancelled by a switch, not by
		// the caller's ctx.
		if err != nil && ctx.Err() == nil {
			continue
		}
		return m, err
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

// SetType builds (if needed) and swaps to the requested queue. Switching to the
// already-active type is a no-op. A build failure returns an error and leaves
// the active queue unchanged. On a real switch the parked Dequeue is released so
// it picks up the new active queue. The queue is built outside the swap lock so
// a slow connect does not block delegation.
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
