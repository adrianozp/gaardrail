package switchcontroller

//go:generate mockery --all --output=mocks --outpkg=mocks

// Controller is the runtime-switchable controller. Satisfied by
// *switchable.Controller.
type Controller interface {
	SetType(t string) error
	Type() string
}

// TypePersister stores the selected controller type so it survives a restart.
type TypePersister interface {
	PersistControllerType(t string) error
}

type UseCase struct {
	controller Controller
	persister  TypePersister
}

func New(c Controller, p TypePersister) UseCase {
	return UseCase{controller: c, persister: p}
}

// Switch changes the active controller and then persists the choice. The
// runtime swap happens first so the change takes effect immediately; a
// persistence failure is reported but the new controller is already active.
func (u UseCase) Switch(t string) error {
	if err := u.controller.SetType(t); err != nil {
		return err
	}
	return u.persister.PersistControllerType(t)
}

func (u UseCase) Current() string {
	return u.controller.Type()
}
