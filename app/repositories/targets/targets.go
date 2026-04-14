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
	Push(context.Context, entities.Message) error
}

func NewTarget(cfg config.Config) (Pusher, error) {
	switch cfg.Target.Protocol {
	case "http":
		client := httpclient.New(httpclient.Config{BaseURL: cfg.Target.BaseURL})
		return httprepo.NewHTTPRepository(client, httprepo.Config{Path: cfg.Target.Path}), nil
	case "sql":
		client, err := sqlclient.New(sqlclient.Config{
			DSN:    cfg.Target.DSN,
			Driver: cfg.Target.Driver,
		})
		if err != nil {
			return nil, fmt.Errorf("target: sql: %w", err)
		}
		return sqlrepo.NewSQLRepository(client), nil
	default:
		return nil, fmt.Errorf("target: unknown protocol %q", cfg.Target.Protocol)
	}
}

// noopTarget is used when MetricsPoller.Enabled is false.
type noopTarget struct{}

func (n *noopTarget) Push(_ context.Context, _ entities.Message) error {
	return nil
}
