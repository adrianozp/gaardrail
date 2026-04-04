package http

import (
	"encoding/json"

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

func (r *HTTPRepository) Push(m entities.Message) error {
	model := FromMesssage(m)
	body, err := json.Marshal(model)
	if err != nil {
		return err
	}
	log.Debug().Bytes("body", body).Msg("sent command body")
	return r.client.Post(r.path, body)
}
