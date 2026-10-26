package controllerparams

import (
	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/rs/zerolog/log"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Controller interface {
	SetParams(p entities.ControllerParams) error
	GetParams() entities.ControllerParams
	Type() string
}

type ConfigStore interface {
	Set(updates map[string]any) error
}

type UseCase struct {
	controller Controller
	store      ConfigStore
}

func New(c Controller, s ConfigStore) UseCase {
	return UseCase{controller: c, store: s}
}

func (u UseCase) Update(p entities.ControllerParams) error {
	if err := u.controller.SetParams(p); err != nil {
		return err
	}
	updates := persistedUpdates(sectionFor(u.controller.Type()), p, u.controller.GetParams())
	if len(updates) == 0 {
		return nil
	}
	if err := u.store.Set(updates); err != nil {
		log.Warn().Err(err).Msg("controllerparams: persist failed")
	}
	return nil
}

func (u UseCase) Get() entities.ControllerParams {
	return u.controller.GetParams()
}

func (u UseCase) CurrentType() string {
	return u.controller.Type()
}

func sectionFor(controllerType string) string {
	if controllerType == "smith" {
		return "smith"
	}
	return "pid"
}

func persistedUpdates(section string, requested, effective entities.ControllerParams) map[string]any {
	asked := fieldValues(requested)
	if requested.SetpointFilterType != nil || requested.SetpointFilterSize != nil {
		asked["setpoint_filter_type"], asked["setpoint_filter_size"] = nil, nil
	}
	updates := map[string]any{}
	for key, value := range fieldValues(effective) {
		if _, ok := asked[key]; ok {
			updates[section+"."+key] = value
		}
	}
	return updates
}

func fieldValues(p entities.ControllerParams) map[string]any {
	values := map[string]any{}
	putIfSet(values, "setpoint", p.Setpoint)
	putIfSet(values, "kp", p.Kp)
	putIfSet(values, "ki", p.Ki)
	putIfSet(values, "kd", p.Kd)
	putIfSet(values, "min", p.Min)
	putIfSet(values, "max", p.Max)
	putIfSet(values, "i_clamp", p.IClamp)
	putIfSet(values, "ff_gain", p.FfGain)
	putIfSet(values, "setpoint_filter_type", p.SetpointFilterType)
	putIfSet(values, "setpoint_filter_size", p.SetpointFilterSize)
	putIfSet(values, "model_k", p.ModelK)
	putIfSet(values, "model_tau", p.ModelTau)
	putIfSet(values, "model_theta", p.ModelTheta)
	putIfSet(values, "sample_seconds", p.SampleSeconds)
	return values
}

func putIfSet[T any](values map[string]any, key string, v *T) {
	if v != nil {
		values[key] = *v
	}
}
