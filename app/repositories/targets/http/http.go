package http

import (
	"context"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/internal/httpclient"
	"github.com/adrianozp/gaardrail/pkg/clock"
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

func (r *HTTPRepository) Push(ctx context.Context, m entities.Message) (entities.Response, error) {
	log.Debug().Str("message_id", m.ID).Msg("sent command body")

	response, err := r.client.Post(ctx, r.path, m.Body)
	if err != nil {
		return entities.Response{}, err
	}

	return entities.Response{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		PushedAt:  clock.Now(),
		Body:      response,
	}, nil
}
