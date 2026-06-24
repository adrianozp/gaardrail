package switchqueue_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/usecases/switchqueue"
	"github.com/adrianozp/gaardrail/app/usecases/switchqueue/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitch_SwapsQueueTypeAndPersists(t *testing.T) {
	q := mocks.NewQueue(t)
	q.On("SetType", "constant").Return(nil)
	store := mocks.NewConfigStore(t)
	store.On("Set", map[string]any{"queue.protocol": "constant"}).Return(nil)

	uc := switchqueue.New(q, store)

	require.NoError(t, uc.Switch("constant"))
}

func TestSwitch_SetTypeErrorIsReturnedAndNotPersisted(t *testing.T) {
	q := mocks.NewQueue(t)
	q.On("SetType", "banana").Return(errors.New("unknown type"))
	store := mocks.NewConfigStore(t)

	uc := switchqueue.New(q, store)

	require.Error(t, uc.Switch("banana"))
	store.AssertNotCalled(t, "Set")
}

func TestSwitch_PersistFailureDoesNotFailSwitch(t *testing.T) {
	q := mocks.NewQueue(t)
	q.On("SetType", "constant").Return(nil)
	store := mocks.NewConfigStore(t)
	store.On("Set", map[string]any{"queue.protocol": "constant"}).Return(errors.New("read-only"))

	uc := switchqueue.New(q, store)

	require.NoError(t, uc.Switch("constant"))
}

func TestCurrent_ReturnsQueueType(t *testing.T) {
	q := mocks.NewQueue(t)
	q.On("Type").Return("inmemory")

	uc := switchqueue.New(q, mocks.NewConfigStore(t))

	assert.Equal(t, "inmemory", uc.Current())
}

func TestAvailable_ReturnsSwitchableTypes(t *testing.T) {
	q := mocks.NewQueue(t)
	q.On("Available").Return([]string{"inmemory", "constant"})

	uc := switchqueue.New(q, mocks.NewConfigStore(t))

	assert.Equal(t, []string{"inmemory", "constant"}, uc.Available())
}
