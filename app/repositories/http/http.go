package http

import (
	"encoding/json"
	"log"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/internal/httpclient"
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
	log.Printf("sent command body %s", body)
	return r.client.Post(r.path, body)
}
