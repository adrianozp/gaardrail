package sql

import (
	"context"
	"fmt"

	"github.com/adrianozp/gaardrail/app/entities"
	sqlclient "github.com/adrianozp/gaardrail/internal/sqlclient"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/rs/zerolog/log"
)

type SQLRepository struct {
	client *sqlclient.Client
}

func NewSQLRepository(client *sqlclient.Client) *SQLRepository {
	return &SQLRepository{client: client}
}

func (r *SQLRepository) Push(ctx context.Context, m entities.Message) (entities.Response, error) {
	log.Debug().Str("message_id", m.ID).Msg("executing sql query")
	rows, err := r.client.ExecContext(ctx, string(m.Body))
	if err != nil {
		return entities.Response{}, err
	}
	return entities.Response{
		ID:        m.ID,
		CreatedAt: m.CreatedAt,
		PushedAt:  clock.Now(),
		Body:      []byte(fmt.Sprintf("%d", rows)),
	}, nil
}
