package options

import (
	"context"

	"github.com/adrianozp/gaardrail/app/usecases/orchestrator"
	"github.com/adrianozp/gaardrail/cmd/api/modules"
	"github.com/adrianozp/gaardrail/internal/httpserver"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func Options() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			httpserver.New,
		),

		modules.KafkaFactories(),
		modules.HTTPFactories(),

		modules.OrchestratorFactories(),
		modules.OrchestratorInjections(),

		modules.MetricsPollerFactories(),
		modules.MetricsPollerInjections(),
		modules.MetricsPollerLifecycle(),

		modules.MessageFactories(),
		modules.MessageInjections(),
		modules.MessageEndpoints(),

		fx.Invoke(func(lc fx.Lifecycle, router *gin.Engine, cfg config.Config) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go router.Run(cfg.HTTP.Addr)
					return nil
				},
			})
		}),

		fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return o.Start(ctx)
				},
			})
		}),
	)
}
