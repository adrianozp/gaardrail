package consumemessage

import "github.com/adrianozp/gaardrail/app/entities"

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
	message, err := u.queue.Dequeue()
	if err != nil {
		return "", err
	}

	err = u.target.Push(message)
	if err != nil {
		return "", err
	}

	err = u.queue.Ack(message)
	if err != nil {
		return "", err
	}

	return message.ID, nil
}

func (u ConsumeMessageUseCase) Size() (int64, error) {
	return u.queue.Size()
}
