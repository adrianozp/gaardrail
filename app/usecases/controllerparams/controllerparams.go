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

// ConfigStore persists runtime changes back to the config file so they survive
// a restart.
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

// Update applies the params at runtime and persists the changed ones to the pid
// section so they are restored on restart. A failed persist is logged but does
// not undo the runtime change.
func (u UseCase) Update(p entities.ControllerParams) error {
	if err := u.controller.SetParams(p); err != nil {
		return err
	}

	updates := pidUpdates(p)
	if p.SetpointFilterType != nil || p.SetpointFilterSize != nil {
		for k, v := range effectiveFilterUpdates(u.controller.GetParams()) {
			updates[k] = v
		}
	}

	if len(updates) > 0 {
		if err := u.store.Set(updates); err != nil {
			log.Warn().Err(err).Msg("controllerparams: persist failed")
		}
	}
	return nil
}

func (u UseCase) Get() entities.ControllerParams {
	return u.controller.GetParams()
}

// CurrentType returns the type of the currently active controller.
func (u UseCase) CurrentType() string {
	return u.controller.Type()
}

// pidUpdates maps the provided (non-nil) params to their dotted config keys.
func pidUpdates(p entities.ControllerParams) map[string]any {
	updates := map[string]any{}
	if p.Setpoint != nil {
		updates["pid.setpoint"] = *p.Setpoint
	}
	if p.Kp != nil {
		updates["pid.kp"] = *p.Kp
	}
	if p.Ki != nil {
		updates["pid.ki"] = *p.Ki
	}
	if p.Kd != nil {
		updates["pid.kd"] = *p.Kd
	}
	if p.Min != nil {
		updates["pid.min"] = *p.Min
	}
	if p.Max != nil {
		updates["pid.max"] = *p.Max
	}
	if p.IClamp != nil {
		updates["pid.i_clamp"] = *p.IClamp
	}
	return updates
}

func effectiveFilterUpdates(effective entities.ControllerParams) map[string]any {
	updates := map[string]any{}
	if effective.SetpointFilterType != nil {
		updates["pid.setpoint_filter_type"] = *effective.SetpointFilterType
	}
	if effective.SetpointFilterSize != nil {
		updates["pid.setpoint_filter_size"] = *effective.SetpointFilterSize
	}
	return updates
}
