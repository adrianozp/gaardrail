package options

import (
	"context"

	"github.com/adrianozp/gaardrail/app/orchestrator"
	"github.com/adrianozp/gaardrail/cmd/api/modules"
	"github.com/adrianozp/gaardrail/internal/httpserver"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

func Options() fx.Option {
	return fx.Options(
		fx.Provide(
			config.Load,
			config.NewPersister,
			httpserver.New,
		),

		modules.MetricsLifecycle(),

		modules.QueueFactories(),
		modules.QueueInjections(),
		modules.QueueEndpoints(),
		modules.QueueLifecycle(),

		modules.OrchestratorFactories(),
		modules.OrchestratorInjections(),

		modules.DisturbanceFactories(),
		modules.DisturbanceInjections(),
		modules.DisturbanceEndpoints(),
		modules.DisturbanceLifecycle(),

		modules.MetricsPollerFactories(),
		modules.MetricsPollerInjections(),
		modules.MetricsPollerLifecycle(),

		modules.MessageFactories(),
		modules.MessageInjections(),
		modules.MessageEndpoints(),

		modules.ControllerFactories(),
		modules.ControllerInjections(),
		modules.ControllerEndpoints(),

		fx.Invoke(func(lc fx.Lifecycle, router *gin.Engine, cfg config.Config) {
			router.GET("/metrics", gin.WrapH(promhttp.Handler()))
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if cfg.HTTP.TLSEnabled() {
						go router.RunTLS(cfg.HTTP.Addr, cfg.HTTP.CertFile, cfg.HTTP.KeyFile)
					} else {
						go router.Run(cfg.HTTP.Addr)
					}
					return nil
				},
			})
		}),

		fx.Invoke(func(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return o.Start(context.Background())
				},
			})
		}),
	)
}
