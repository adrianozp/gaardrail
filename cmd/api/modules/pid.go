package modules

import (
	pidparamshandler "github.com/adrianozp/gaardrail/app/handlers/pidparams"
	"github.com/adrianozp/gaardrail/app/usecases/pidparams"
	"github.com/adrianozp/gaardrail/internal/controller"
	"go.uber.org/fx"
)

func PIDFactories() fx.Option {
	return fx.Provide(
		pidparams.New,
		pidparamshandler.New,
	)
}

func PIDInjections() fx.Option {
	return fx.Provide(
		func(c *controller.Controller) pidparams.Controller { return c },
		func(uc pidparams.UseCase) pidparamshandler.PIDParamsUseCase { return uc },
	)
}

func PIDEndpoints() fx.Option {
	return fx.Module("pid",
		fx.Invoke(pidparamshandler.RegisterRoutes),
	)
}
