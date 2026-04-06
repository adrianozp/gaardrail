package pidparams

import "github.com/adrianozp/gaardrail/app/entities"

//go:generate mockery --all --output=mocks --outpkg=mocks
type Controller interface {
	SetParams(p entities.PIDParams) error
	GetParams() entities.PIDParams
}

type UseCase struct {
	controller Controller
}

func New(c Controller) UseCase {
	return UseCase{controller: c}
}

func (u UseCase) Update(p entities.PIDParams) error {
	return u.controller.SetParams(p)
}

func (u UseCase) Get() entities.PIDParams {
	return u.controller.GetParams()
}
