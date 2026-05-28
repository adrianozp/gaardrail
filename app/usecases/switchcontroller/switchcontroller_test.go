package switchcontroller_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller"
	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitch_SwapsControllerType(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetType", "step").Return(nil)

	uc := switchcontroller.New(ctrl)

	require.NoError(t, uc.Switch("step"))
}

func TestSwitch_SetTypeErrorIsReturned(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetType", "banana").Return(errors.New("unknown type"))

	uc := switchcontroller.New(ctrl)

	require.Error(t, uc.Switch("banana"))
}

func TestCurrent_ReturnsControllerType(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("Type").Return("pid")

	uc := switchcontroller.New(ctrl)

	assert.Equal(t, "pid", uc.Current())
}
