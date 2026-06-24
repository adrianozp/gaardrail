package switchable

import (
	"context"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
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

func TestNew_SelectsConfiguredQueue(t *testing.T) {
	q := newQueue(t, "constant")
	assert.Equal(t, "constant", q.Type())
}

func TestNew_UnknownTypeErrors(t *testing.T) {
	cfg := config.Config{Queue: config.Queue{Protocol: "kafka"}}
	_, err := New(cfg, constant.New(cfg))
	require.Error(t, err)
}

func TestAvailable_ListsSwitchableQueues(t *testing.T) {
	q := newQueue(t, "inmemory")
	assert.Equal(t, []string{"inmemory", "constant"}, q.Available())
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

	time.Sleep(20 * time.Millisecond) // let the goroutine park on the empty queue
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
