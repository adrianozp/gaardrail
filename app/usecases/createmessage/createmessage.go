package createmessage

import "github.com/adrianozp/gaardrail/app/entities"

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
	id, err := u.queue.Enqueue(m)
	if err != nil {
		return "", err
	}

	return id, err
}
