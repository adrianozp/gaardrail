package switchcontroller_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller"
	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitch_SwapsControllerTypeAndPersists(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetType", "step").Return(nil)
	store := mocks.NewConfigStore(t)
	store.On("Set", map[string]any{"controller.type": "step"}).Return(nil)

	uc := switchcontroller.New(ctrl, store)

	require.NoError(t, uc.Switch("step"))
}

func TestSwitch_SetTypeErrorIsReturnedAndNotPersisted(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetType", "banana").Return(errors.New("unknown type"))
	store := mocks.NewConfigStore(t)

	uc := switchcontroller.New(ctrl, store)

	require.Error(t, uc.Switch("banana"))
	store.AssertNotCalled(t, "Set")
}

func TestCurrent_ReturnsControllerType(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("Type").Return("pid")

	uc := switchcontroller.New(ctrl, mocks.NewConfigStore(t))

	assert.Equal(t, "pid", uc.Current())
}
