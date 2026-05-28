package modules

import (
	controllerparamshandler "github.com/adrianozp/gaardrail/app/handlers/controllerparams"
	switchcontrollerhandler "github.com/adrianozp/gaardrail/app/handlers/switchcontroller"
	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/switchable"
	"github.com/adrianozp/gaardrail/app/usecases/controllerparams"
	"github.com/adrianozp/gaardrail/app/usecases/switchcontroller"
	"go.uber.org/fx"
)

func ControllerFactories() fx.Option {
	return fx.Provide(
		controllerparams.New,
		controllerparamshandler.New,
		switchcontroller.New,
		switchcontrollerhandler.New,
		switchable.New,
	)
}

func ControllerInjections() fx.Option {
	return fx.Provide(
		func(c *switchable.Controller) controllers.Controller { return c },
		func(c *switchable.Controller) switchcontroller.Controller { return c },
		func(c controllers.Controller) controllerparams.Controller { return c },
		func(uc controllerparams.UseCase) controllerparamshandler.ControllerParamsUseCase { return uc },
		func(uc switchcontroller.UseCase) switchcontrollerhandler.SwitchControllerUseCase { return uc },
	)
}

func ControllerEndpoints() fx.Option {
	return fx.Module("controller",
		fx.Invoke(controllerparamshandler.RegisterRoutes),
		fx.Invoke(switchcontrollerhandler.RegisterRoutes),
	)
}
