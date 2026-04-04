package createmessage

import (
	"github.com/adrianozp/gaardrail/app/entities"
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
	id, err := u.queue.Enqueue(m)
	if err != nil {
		return "", err
	}

	log.Info().Str("message_id", id).Msg("created message")
	return id, err
}
