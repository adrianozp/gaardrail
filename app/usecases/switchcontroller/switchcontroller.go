package switchcontroller

//go:generate mockery --all --output=mocks --outpkg=mocks

// Controller is the runtime-switchable controller. Satisfied by
// *switchable.Controller.
type Controller interface {
	SetType(t string) error
	Type() string
}

type UseCase struct {
	controller Controller
}

func New(c Controller) UseCase {
	return UseCase{controller: c}
}

// Switch changes the active controller at runtime. The choice is not persisted
// to the config file: in deployment the config is a read-only volume, so the
// switch lives only for the process lifetime and resets to the configured type
// on restart.
func (u UseCase) Switch(t string) error {
	return u.controller.SetType(t)
}

func (u UseCase) Current() string {
	return u.controller.Type()
}
