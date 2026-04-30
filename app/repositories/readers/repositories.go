package readers

import (
	"context"

	"github.com/adrianozp/gaardrail/app/entities"
)

type MetricsReader interface {
	Read(ctx context.Context) (entities.Metrics, error)
}

type Noop struct{}

func (Noop) Read(_ context.Context) (entities.Metrics, error) {
	return entities.Metrics{}, nil
}
