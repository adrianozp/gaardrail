package modules

import (
	"fmt"

	controllerparamshandler "github.com/adrianozp/gaardrail/app/handlers/controllerparams"
	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/pid"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/step"
	"github.com/adrianozp/gaardrail/app/usecases/controllerparams"
	"github.com/adrianozp/gaardrail/pkg/config"
	"go.uber.org/fx"
)

func newController(cfg config.Config) (controllers.Controller, error) {
	switch cfg.Controller.Type {
	case "pid":
		return pid.New(cfg), nil
	case "step":
		return step.NewStep(cfg), nil
	default:
		return nil, fmt.Errorf("controller: unknown type %q", cfg.Controller.Type)
	}
}

func ControllerFactories() fx.Option {
	return fx.Provide(
		controllerparams.New,
		controllerparamshandler.New,
		newController,
	)
}

func ControllerInjections() fx.Option {
	return fx.Provide(
		func(c controllers.Controller) controllerparams.Controller { return c },
		func(uc controllerparams.UseCase) controllerparamshandler.ControllerParamsUseCase { return uc },
	)
}

func ControllerEndpoints() fx.Option {
	return fx.Module("controller",
		fx.Invoke(controllerparamshandler.RegisterRoutes),
	)
}
