package controllerparams_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/controllerparams"
	"github.com/adrianozp/gaardrail/app/usecases/controllerparams/mocks"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func controller(t *testing.T, kind string, requested, effective entities.ControllerParams) *mocks.Controller {
	t.Helper()
	ctrl := mocks.NewController(t)
	ctrl.On("SetParams", requested).Return(nil)
	ctrl.On("GetParams").Return(effective).Maybe()
	ctrl.On("Type").Return(kind).Maybe()
	return ctrl
}

func expectPersisted(t *testing.T, kind string, requested, effective entities.ControllerParams, want map[string]any) {
	t.Helper()
	store := mocks.NewConfigStore(t)
	store.On("Set", want).Return(nil)
	uc := controllerparams.New(controller(t, kind, requested, effective), store)
	require.NoError(t, uc.Update(requested))
}

func TestUpdate_AppliesAndPersistsChangedParams(t *testing.T) {
	p := entities.ControllerParams{Setpoint: ptr(72.0), Kp: ptr(0.5)}
	effective := entities.ControllerParams{Setpoint: ptr(72.0), Kp: ptr(0.5), Ki: ptr(0.1)}
	expectPersisted(t, "pidff", p, effective, map[string]any{"pid.setpoint": 72.0, "pid.kp": 0.5})
}

func TestUpdate_PersistsSetpointFilterTypeAndSize(t *testing.T) {
	p := entities.ControllerParams{SetpointFilterType: ptr("moving_average"), SetpointFilterSize: ptr(4)}
	expectPersisted(t, "pid", p, p, map[string]any{"pid.setpoint_filter_type": "moving_average", "pid.setpoint_filter_size": 4})
}

func TestUpdate_FilterTypeOnly_PersistsEffectiveTypeAndSize(t *testing.T) {
	p := entities.ControllerParams{SetpointFilterType: ptr("moving_average")}
	effective := entities.ControllerParams{SetpointFilterType: ptr("moving_average"), SetpointFilterSize: ptr(2)}
	expectPersisted(t, "pid", p, effective, map[string]any{"pid.setpoint_filter_type": "moving_average", "pid.setpoint_filter_size": 2})
}

func TestUpdate_FilterSizeOnly_PersistsEffectiveTypeAndSize(t *testing.T) {
	p := entities.ControllerParams{SetpointFilterSize: ptr(4)}
	effective := entities.ControllerParams{SetpointFilterType: ptr("exponential"), SetpointFilterSize: ptr(4)}
	expectPersisted(t, "pid", p, effective, map[string]any{"pid.setpoint_filter_type": "exponential", "pid.setpoint_filter_size": 4})
}

func TestUpdate_UnrelatedFieldOnly_FilterKeysNotPersisted(t *testing.T) {
	p := entities.ControllerParams{Kp: ptr(0.5)}
	effective := entities.ControllerParams{Kp: ptr(0.5), SetpointFilterType: ptr("none"), SetpointFilterSize: ptr(1)}
	expectPersisted(t, "pid", p, effective, map[string]any{"pid.kp": 0.5})
}

func TestUpdate_SmithParamsPersistToSmithSection(t *testing.T) {
	p := entities.ControllerParams{Kp: ptr(0.2), ModelTheta: ptr(60.0)}
	effective := entities.ControllerParams{Kp: ptr(0.2), ModelTheta: ptr(60.0)}
	expectPersisted(t, "smith", p, effective, map[string]any{"smith.kp": 0.2, "smith.model_theta": 60.0})
}

func TestUpdate_FeedforwardGainIsPersisted(t *testing.T) {
	p := entities.ControllerParams{FfGain: ptr(2.4)}
	expectPersisted(t, "pidff", p, p, map[string]any{"pid.ff_gain": 2.4})
}

func TestUpdate_FieldsTheControllerIgnoresAreNotPersisted(t *testing.T) {
	p := entities.ControllerParams{Kd: ptr(1.0), Max: ptr(30.0)}
	effective := entities.ControllerParams{Max: ptr(30.0)}
	expectPersisted(t, "smith", p, effective, map[string]any{"smith.max": 30.0})
}

func TestUpdate_NothingToPersistSkipsStore(t *testing.T) {
	p := entities.ControllerParams{Kd: ptr(1.0)}
	store := mocks.NewConfigStore(t)
	uc := controllerparams.New(controller(t, "step", p, entities.ControllerParams{Max: ptr(5.0)}), store)
	require.NoError(t, uc.Update(p))
	store.AssertNotCalled(t, "Set")
}

func TestUpdate_SetParamsErrorIsReturnedAndNotPersisted(t *testing.T) {
	ctrl := mocks.NewController(t)
	ctrl.On("SetParams", entities.ControllerParams{Max: ptr(5.0)}).Return(errors.New("bad"))
	store := mocks.NewConfigStore(t)
	uc := controllerparams.New(ctrl, store)
	require.Error(t, uc.Update(entities.ControllerParams{Max: ptr(5.0)}))
	store.AssertNotCalled(t, "Set")
}
