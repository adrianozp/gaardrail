package options

import (
	"context"

	"github.com/adrianozp/gaardrail/app/orchestrator"
	"github.com/adrianozp/gaardrail/cmd/api/modules"
	"github.com/adrianozp/gaardrail/internal/httpserver"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func Options() fx.Option {
	return fx.Options(
		fx.Provide(config.Load, config.NewPersister, httpserver.New),
		modules.MetricsLifecycle(),
		queueModules(),
		modules.OrchestratorFactories(),
		modules.OrchestratorInjections(),
		metricsPollerModules(),
		messageModules(),
		controllerModules(),
		fx.Invoke(serveHTTP),
		fx.Invoke(runOrchestrator),
	)
}

func queueModules() fx.Option {
	return fx.Options(
		modules.QueueFactories(),
		modules.QueueInjections(),
		modules.QueueEndpoints(),
		modules.QueueLifecycle(),
	)
}

func metricsPollerModules() fx.Option {
	return fx.Options(
		modules.MetricsPollerFactories(),
		modules.MetricsPollerInjections(),
		modules.MetricsPollerLifecycle(),
	)
}

func messageModules() fx.Option {
	return fx.Options(
		modules.MessageFactories(),
		modules.MessageInjections(),
		modules.MessageEndpoints(),
	)
}

func controllerModules() fx.Option {
	return fx.Options(
		modules.ControllerFactories(),
		modules.ControllerInjections(),
		modules.ControllerEndpoints(),
	)
}

func serveHTTP(lc fx.Lifecycle, router *gin.Engine, cfg config.Config) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	lc.Append(fx.StartHook(func() { go runRouter(router, cfg.HTTP) }))
}

func runRouter(router *gin.Engine, http config.HTTP) {
	if http.TLSEnabled() {
		logStopped(router.RunTLS(http.Addr, http.CertFile, http.KeyFile), "options: http server (tls) stopped")
		return
	}
	logStopped(router.Run(http.Addr), "options: http server stopped")
}

func logStopped(err error, msg string) {
	if err != nil {
		log.Error().Err(err).Msg(msg)
	}
}

func runOrchestrator(lc fx.Lifecycle, o *orchestrator.Orchestrator) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error { return o.Start(context.Background()) },
		OnStop:  o.Stop,
	})
}
