package switchcontroller

import "github.com/rs/zerolog/log"

//go:generate mockery --all --output=mocks --outpkg=mocks

// Controller is the runtime-switchable controller. Satisfied by
// *switchable.Controller.
type Controller interface {
	SetType(t string) error
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

// Switch changes the active controller at runtime and persists the choice so it
// is restored on restart. A failed persist is logged but does not undo the
// runtime switch.
func (u UseCase) Switch(t string) error {
	if err := u.controller.SetType(t); err != nil {
		return err
	}

	if err := u.store.Set(map[string]any{"controller.type": t}); err != nil {
		log.Warn().Err(err).Msg("switchcontroller: persist failed")
	}
	return nil
}

func (u UseCase) Current() string {
	return u.controller.Type()
}
