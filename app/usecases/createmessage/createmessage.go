package createmessage

import (
	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/adrianozp/gaardrail/pkg/uuid"
	"github.com/rs/zerolog/log"
)

//go:generate mockery --all --output=mocks --outpkg=mocks

type Queue interface {
	Enqueue(entities.Message) (string, error)
}

type CreateMessageUseCase struct {
	queue Queue
}

func NewCreateMessageUseCase(q Queue) CreateMessageUseCase {
	return CreateMessageUseCase{
		queue: q,
	}
}

func (u CreateMessageUseCase) Create(m entities.Message) (string, error) {
	m.ID = uuid.New()
	enqueueStart := clock.Now()
	id, err := u.queue.Enqueue(m)
	metrics.Gauge(map[string]float64{"enqueue_time_seconds": clock.Now().Sub(enqueueStart).Seconds()})
	if err != nil {
		return "", err
	}

	log.Info().Str("message_id", id).Msg("created message")
	return id, err
}
