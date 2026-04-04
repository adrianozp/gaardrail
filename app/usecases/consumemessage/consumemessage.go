package consumemessage

import (
	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/rs/zerolog/log"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Queue interface {
	Dequeue() (entities.Message, error)
	Ack(entities.Message) error
	Size() (int64, error)
}

type Target interface {
	Push(entities.Message) error
}

type ConsumeMessageUseCase struct {
	queue  Queue
	target Target
}

func NewConsumeMessageUseCase(q Queue, t Target) ConsumeMessageUseCase {
	return ConsumeMessageUseCase{
		queue:  q,
		target: t,
	}
}

func (u ConsumeMessageUseCase) Consume() (string, error) {
	startTime := clock.Now()
	message, err := u.queue.Dequeue()
	if err != nil {
		log.Error().Msg("error dequeing message")
		return "", err
	}

	err = u.target.Push(message)
	if err != nil {
		log.Error().Str("message_id", message.ID).Msg("error pushing message")
		return "", err
	}

	err = u.queue.Ack(message)
	if err != nil {
		log.Error().Str("message_id", message.ID).Msg("error consuming message")
		return "", err
	}

	elapsedTime := clock.Now().Sub(startTime).Seconds()
	log.Info().Str("message_id", message.ID).Float64("elapsed_time", elapsedTime).Msg("consumed message")
	return message.ID, nil
}

func (u ConsumeMessageUseCase) Size() (int64, error) {
	return u.queue.Size()
}
