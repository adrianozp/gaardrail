package switchable

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/queue"
	"github.com/adrianozp/gaardrail/app/repositories/queue/constant"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newQueue(t *testing.T, active string) *Queue {
	t.Helper()
	cfg := config.Config{Queue: config.Queue{Protocol: active, Capacity: 10, Query: "SELECT 1"}}
	q, err := New(cfg, constant.New(cfg))
	require.NoError(t, err)
	return q
}

// fakeQueue is a minimal queue whose Dequeue blocks until the context is done,
// used to drive lazy-construction tests without external dependencies.
type fakeQueue struct{ name string }

func (f *fakeQueue) Enqueue(entities.Message) (string, error) { return "", nil }
func (f *fakeQueue) Dequeue(ctx context.Context) (entities.Message, error) {
	<-ctx.Done()
	return entities.Message{}, ctx.Err()
}
func (f *fakeQueue) Ack(context.Context, entities.Message) error { return nil }
func (f *fakeQueue) Size() (int64, error)                        { return 0, nil }

// rawQueue builds a Queue with custom constructors for white-box lazy tests.
func rawQueue(t *testing.T, active string, ctors map[string]func() (queue.Queue, error)) *Queue {
	t.Helper()
	q := &Queue{
		switched:     make(chan struct{}),
		built:        map[string]queue.Queue{},
		constructors: ctors,
	}
	inst, err := q.instance(active)
	require.NoError(t, err)
	q.active = inst
	q.activeType = active
	return q
}

func TestNew_SelectsConfiguredQueue(t *testing.T) {
	q := newQueue(t, "constant")
	assert.Equal(t, "constant", q.Type())
}

func TestNew_UnknownTypeErrors(t *testing.T) {
	cfg := config.Config{Queue: config.Queue{Protocol: "banana"}}
	_, err := New(cfg, constant.New(cfg))
	require.Error(t, err)
}

func TestAvailable_ListsAllSwitchableQueues(t *testing.T) {
	q := newQueue(t, "inmemory")
	assert.Equal(t, []string{"kafka", "inmemory", "constant", "sqs"}, q.Available())
}

func TestSetType_SwitchesActive(t *testing.T) {
	q := newQueue(t, "inmemory")
	require.NoError(t, q.SetType("constant"))
	assert.Equal(t, "constant", q.Type())
}

func TestSetType_SameTypeIsNoOp(t *testing.T) {
	q := newQueue(t, "inmemory")
	require.NoError(t, q.SetType("inmemory"))
	assert.Equal(t, "inmemory", q.Type())
}

func TestSetType_UnknownTypeErrorsAndKeepsActive(t *testing.T) {
	q := newQueue(t, "inmemory")
	require.Error(t, q.SetType("banana"))
	assert.Equal(t, "inmemory", q.Type(), "active queue must be unchanged on error")
}

func TestSetType_BuildsLazilyAndCaches(t *testing.T) {
	calls := 0
	ctors := map[string]func() (queue.Queue, error){
		"a": func() (queue.Queue, error) { return &fakeQueue{"a"}, nil },
		"b": func() (queue.Queue, error) { calls++; return &fakeQueue{"b"}, nil },
	}
	q := rawQueue(t, "a", ctors)

	require.NoError(t, q.SetType("b"))
	require.NoError(t, q.SetType("a"))
	require.NoError(t, q.SetType("b"))

	assert.Equal(t, 1, calls, "b must be constructed once and reused from cache")
}

func TestSetType_ConstructorErrorKeepsActive(t *testing.T) {
	ctors := map[string]func() (queue.Queue, error){
		"a":   func() (queue.Queue, error) { return &fakeQueue{"a"}, nil },
		"bad": func() (queue.Queue, error) { return nil, errors.New("connect failed") },
	}
	q := rawQueue(t, "a", ctors)

	require.Error(t, q.SetType("bad"), "a failed construction must surface as an error")
	assert.Equal(t, "a", q.Type(), "active queue must be unchanged when construction fails")
}

func TestEnqueueSize_DelegateToActive(t *testing.T) {
	q := newQueue(t, "inmemory")

	_, err := q.Enqueue(entities.Message{ID: "a"})
	require.NoError(t, err)

	size, err := q.Size()
	require.NoError(t, err)
	assert.Equal(t, int64(1), size, "inmemory must report its buffered size")

	require.NoError(t, q.SetType("constant"))

	size, err = q.Size()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), size, "constant reports infinite size (-1)")
}

// A worker blocked on an empty inmemory Dequeue must be released when the queue
// is switched, then resume against the newly active queue.
func TestDequeue_UnparkedOnSwitch(t *testing.T) {
	q := newQueue(t, "inmemory")

	done := make(chan string, 1)
	go func() {
		m, err := q.Dequeue(context.Background())
		if err != nil {
			done <- "err:" + err.Error()
			return
		}
		done <- string(m.Body)
	}()

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, q.SetType("constant"))

	select {
	case body := <-done:
		assert.Equal(t, "SELECT 1", body)
	case <-time.After(time.Second):
		t.Fatal("Dequeue was not unparked after switch")
	}
}

func TestDequeue_RespectsContextCancellation(t *testing.T) {
	q := newQueue(t, "inmemory")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := q.Dequeue(ctx)
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("Dequeue did not return after context cancellation")
	}
}
