package controllerparams

import "github.com/adrianozp/gaardrail/app/entities"

//go:generate mockery --all --output=mocks --outpkg=mocks
type Controller interface {
	SetParams(p entities.ControllerParams) error
	GetParams() entities.ControllerParams
	Type() string
}

type UseCase struct {
	controller Controller
}

func New(c Controller) UseCase {
	return UseCase{controller: c}
}

func (u UseCase) Update(p entities.ControllerParams) error {
	return u.controller.SetParams(p)
}

func (u UseCase) Get() entities.ControllerParams {
	return u.controller.GetParams()
}

// CurrentType returns the type of the currently active controller.
func (u UseCase) CurrentType() string {
	return u.controller.Type()
}
