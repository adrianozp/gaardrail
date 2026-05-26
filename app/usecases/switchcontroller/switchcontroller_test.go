package switchcontroller_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller"
	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitch_SwapsThenPersists(t *testing.T) {
	ctrl := mocks.NewController(t)
	persister := mocks.NewTypePersister(t)
	ctrl.On("SetType", "step").Return(nil)
	persister.On("PersistControllerType", "step").Return(nil)

	uc := switchcontroller.New(ctrl, persister)

	require.NoError(t, uc.Switch("step"))
}

func TestSwitch_SetTypeErrorSkipsPersist(t *testing.T) {
	ctrl := mocks.NewController(t)
	persister := mocks.NewTypePersister(t)
	ctrl.On("SetType", "banana").Return(errors.New("unknown type"))

	uc := switchcontroller.New(ctrl, persister)

	require.Error(t, uc.Switch("banana"))
	persister.AssertNotCalled(t, "PersistControllerType")
}

func TestSwitch_PersistErrorIsReturned(t *testing.T) {
	ctrl := mocks.NewController(t)
	persister := mocks.NewTypePersister(t)
	ctrl.On("SetType", "step").Return(nil)
	persister.On("PersistControllerType", "step").Return(errors.New("disk full"))

	uc := switchcontroller.New(ctrl, persister)

	require.Error(t, uc.Switch("step"))
}

func TestCurrent_ReturnsControllerType(t *testing.T) {
	ctrl := mocks.NewController(t)
	persister := mocks.NewTypePersister(t)
	ctrl.On("Type").Return("pid")

	uc := switchcontroller.New(ctrl, persister)

	assert.Equal(t, "pid", uc.Current())
}
