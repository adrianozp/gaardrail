package controllerparams_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/controllerparams"
	"github.com/adrianozp/gaardrail/app/usecases/controllerparams/mocks"
	"github.com/stretchr/testify/require"
)

func ptr(f float64) *float64 { return &f }

func TestUpdate_AppliesAndPersistsChangedParams(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetParams", entities.ControllerParams{Setpoint: ptr(72), Kp: ptr(0.5)}).Return(nil)
	store := mocks.NewConfigStore(t)
	store.On("Set", map[string]any{"pid.setpoint": 72.0, "pid.kp": 0.5}).Return(nil)

	uc := controllerparams.New(ctrl, store)

	require.NoError(t, uc.Update(entities.ControllerParams{Setpoint: ptr(72), Kp: ptr(0.5)}))
}

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }

func TestUpdate_PersistsSetpointFilterTypeAndSize(t *testing.T) {
	params := entities.ControllerParams{SetpointFilterType: strptr("moving_average"), SetpointFilterSize: intptr(4)}

	ctrl := mocks.NewController(t)
	ctrl.On("SetParams", params).Return(nil)
	store := mocks.NewConfigStore(t)
	store.On("Set", map[string]any{"pid.setpoint_filter_type": "moving_average", "pid.setpoint_filter_size": 4}).Return(nil)

	uc := controllerparams.New(ctrl, store)

	require.NoError(t, uc.Update(params))
}

func TestUpdate_SetParamsErrorIsReturnedAndNotPersisted(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetParams", entities.ControllerParams{Max: ptr(5)}).Return(errors.New("bad"))
	store := mocks.NewConfigStore(t)

	uc := controllerparams.New(ctrl, store)

	require.Error(t, uc.Update(entities.ControllerParams{Max: ptr(5)}))
	store.AssertNotCalled(t, "Set")
}
