package modules

import (
	updatepidparamshandler "github.com/adrianozp/gaardrail/app/handlers/updatepidparams"
	"github.com/adrianozp/gaardrail/app/usecases/updatepidparams"
	"github.com/adrianozp/gaardrail/internal/controller"
	"go.uber.org/fx"
)

func PIDFactories() fx.Option {
	return fx.Provide(
		updatepidparams.New,
		updatepidparamshandler.NewUpdatePIDParamsHandler,
	)
}

func PIDInjections() fx.Option {
	return fx.Provide(
		func(c *controller.Controller) updatepidparams.ParamUpdater { return c },
		func(uc updatepidparams.UpdatePIDParamsUseCase) updatepidparamshandler.UpdatePIDParamsUseCase {
			return uc
		},
	)
}

func PIDEndpoints() fx.Option {
	return fx.Module("pid",
		fx.Invoke(updatepidparamshandler.RegisterUpdatePIDParamsRoutes),
	)
}
