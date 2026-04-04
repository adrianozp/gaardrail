package updatepidparams

import "github.com/adrianozp/gaardrail/app/entities"

//go:generate mockery --all --output=mocks --outpkg=mocks
type ParamUpdater interface {
	SetParams(p entities.PIDParams) error
}

type UpdatePIDParamsUseCase struct {
	controller ParamUpdater
}

func New(c ParamUpdater) UpdatePIDParamsUseCase {
	return UpdatePIDParamsUseCase{controller: c}
}

func (u UpdatePIDParamsUseCase) Update(p entities.PIDParams) error {
	return u.controller.SetParams(p)
}
