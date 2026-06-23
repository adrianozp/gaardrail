package modules

import (
	"context"

	"github.com/adrianozp/gaardrail/app/disturbance"
	disturbancehandler "github.com/adrianozp/gaardrail/app/handlers/disturbance"
	"github.com/adrianozp/gaardrail/app/usecases/setdisturbance"
	"github.com/adrianozp/gaardrail/internal/sqlclient"
	"go.uber.org/fx"
)

func DisturbanceFactories() fx.Option {
	return fx.Provide(
		disturbance.New,
		setdisturbance.New,
		disturbancehandler.New,
	)
}

func DisturbanceInjections() fx.Option {
	return fx.Provide(
		func(c *sqlclient.Client) disturbance.Executor {
			if c == nil {
				return nil
			}
			return c
		},
		func(d *disturbance.Disturbance) setdisturbance.Component { return d },
		func(uc setdisturbance.UseCase) disturbancehandler.DisturbanceUseCase { return uc },
	)
}

func DisturbanceEndpoints() fx.Option {
	return fx.Module("disturbance",
		fx.Invoke(disturbancehandler.RegisterRoutes),
	)
}

func DisturbanceLifecycle() fx.Option {
	return fx.Invoke(func(lc fx.Lifecycle, d *disturbance.Disturbance) {
		lc.Append(fx.Hook{
			OnStart: func(_ context.Context) error {
				return d.Start(context.Background())
			},
		})
	})
}
