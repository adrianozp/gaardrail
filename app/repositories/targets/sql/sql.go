package sql

import (
	"github.com/adrianozp/gaardrail/app/entities"
	sqlclient "github.com/adrianozp/gaardrail/internal/sqlclient"
	"github.com/rs/zerolog/log"
)

type SQLRepository struct {
	client *sqlclient.Client
}

func NewSQLRepository(client *sqlclient.Client) *SQLRepository {
	return &SQLRepository{client: client}
}

func (r *SQLRepository) Push(m entities.Message) error {
	log.Debug().Str("message_id", m.ID).Msg("executing sql query")
	return r.client.Exec(string(m.Body))
}
