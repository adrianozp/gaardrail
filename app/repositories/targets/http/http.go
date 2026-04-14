package http

import (
	"context"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/internal/httpclient"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Path string
}

type HTTPRepository struct {
	client *httpclient.Client
	path   string
}

func NewHTTPRepository(client *httpclient.Client, cfg Config) *HTTPRepository {
	return &HTTPRepository{
		client: client,
		path:   cfg.Path,
	}
}

func (r *HTTPRepository) Push(ctx context.Context, m entities.Message) error {
	log.Debug().Str("message_id", m.ID).Msg("sent command body")
	return r.client.Post(ctx, r.path, m.Body)
}
