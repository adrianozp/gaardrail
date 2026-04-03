package modules

import (
	httprepo "github.com/adrianozp/gaardrail/app/repositories/http"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/internal/httpclient"
	"go.uber.org/fx"
)

func HTTPFactories() fx.Option {
	return fx.Provide(
		func(cfg config.Config) *httpclient.Client {
			return httpclient.New(httpclient.Config{BaseURL: cfg.Target.BaseURL})
		},
		func(cfg config.Config, client *httpclient.Client) *httprepo.HTTPRepository {
			return httprepo.NewHTTPRepository(client, httprepo.Config{Path: cfg.Target.Path})
		},
	)
}
