package targets

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/entities"
	httprepo "github.com/adrianozp/gaardrail/app/repositories/targets/http"
	sqlrepo "github.com/adrianozp/gaardrail/app/repositories/targets/sql"
	"github.com/adrianozp/gaardrail/internal/httpclient"
	"github.com/adrianozp/gaardrail/internal/sqlclient"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type Pusher interface {
	Push(context.Context, entities.Message) (entities.Response, error)
}

func NewTarget(cfg config.Config, client *sqlclient.Client) (Pusher, error) {
	switch cfg.Target.Protocol {
	case "http":
		httpClient := httpclient.New(httpclient.Config{BaseURL: cfg.Target.BaseURL})
		return httprepo.NewHTTPRepository(httpClient, httprepo.Config{Path: cfg.Target.Path}), nil
	case "sql":
		if client == nil {
			return nil, fmt.Errorf("target: sql: nil sql client")
		}
		return sqlrepo.NewSQLRepository(client), nil
	default:
		return nil, fmt.Errorf("target: unknown protocol %q", cfg.Target.Protocol)
	}
}

// NewSQLClient builds the SQL client shared by the SQL target and the
// disturbance component. It returns nil when the target is not SQL.
func NewSQLClient(cfg config.Config) (*sqlclient.Client, error) {
	if cfg.Target.Protocol != "sql" {
		return nil, nil
	}
	client, err := sqlclient.New(sqlclient.Config{
		DSN:    cfg.Target.DSN,
		Driver: cfg.Target.Driver,
		TLS: sqlclient.TLSConfig{
			Enabled:    cfg.Target.TLS.Enabled,
			CAFile:     cfg.Target.TLS.CAFile,
			SkipVerify: cfg.Target.TLS.SkipVerify,
			ServerName: cfg.Target.TLS.ServerName,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("target: sql: %w", err)
	}
	return client, nil
}
