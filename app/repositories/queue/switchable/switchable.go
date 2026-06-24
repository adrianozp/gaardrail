// Package switchable wraps the queues that can be selected at runtime and lets
// the active one be swapped without restarting. It implements queue.Queue by
// delegating every call to the currently selected queue, so consumers never
// know the underlying implementation changed. The switch logic and all edge
// cases live here, keeping use cases and FX modules free of branching.
package switchable

import (
	"context"
	"fmt"
	"sync"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/repositories/queue/constant"
	"github.com/adrianozp/gaardrail/app/repositories/queue/inmemory"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// availableTypes is the fixed set of runtime-switchable queues. Only queues with
// no external dependencies are switchable; kafka/sqs are selected at boot only.
var availableTypes = []string{"inmemory", "constant"}

// Queue delegates to whichever underlying queue is currently active. Each
// underlying instance stays alive across switches and keeps its own state.
type Queue struct {
	mu         sync.RWMutex
	active     queue.Queue
	activeType string
	available  map[string]queue.Queue
	// switched is closed on every SetType to release a Dequeue parked on the
	// previous (possibly blocking) queue; it is then replaced for the next switch.
	switched chan struct{}
}

// New builds the switchable queues and selects the one named by
// cfg.Queue.Protocol. The constant queue is passed in so it can be shared with
// the queue-query holder. Returns an error for a non-switchable type.
func New(cfg config.Config, cq *constant.Queue) (*Queue, error) {
	available := map[string]queue.Queue{
		"inmemory": inmemory.New(cfg),
		"constant": cq,
	}

	active, ok := available[cfg.Queue.Protocol]
	if !ok {
		return nil, fmt.Errorf("queue: unknown switchable type %q", cfg.Queue.Protocol)
	}

	return &Queue{
		active:     active,
		activeType: cfg.Queue.Protocol,
		available:  available,
		switched:   make(chan struct{}),
	}, nil
}

func (q *Queue) Enqueue(m entities.Message) (string, error) {
	q.mu.RLock()
	active := q.active
	q.mu.RUnlock()
	return active.Enqueue(m)
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
	q.mu.RLock()
	active := q.active
	q.mu.RUnlock()
	return active.Ack(ctx, m)
}

func (q *Queue) Size() (int64, error) {
	q.mu.RLock()
	active := q.active
	q.mu.RUnlock()
	return active.Size()
}

func (q *Queue) Type() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.activeType
}

func (q *Queue) Available() []string {
	return availableTypes
}

// SetType swaps the active queue. Switching to the already-active type is a
// no-op. Unknown types return an error. On a real switch, the parked Dequeue is
// released so it picks up the new active queue.
func (q *Queue) SetType(t string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if t == q.activeType {
		return nil
	}

	next, ok := q.available[t]
	if !ok {
		return fmt.Errorf("queue: unknown type %q", t)
	}

	q.active = next
	q.activeType = t
	close(q.switched)
	q.switched = make(chan struct{})
	return nil
}
