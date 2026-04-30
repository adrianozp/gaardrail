package sqs

import (
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
)

type sqsMessage struct {
	ID        string    `json:"id"`
	Body      []byte    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (s sqsMessage) ToMessage() entities.Message {
	return entities.Message{
		ID:        s.ID,
		Body:      s.Body,
		CreatedAt: s.CreatedAt,
	}
}

func ToModel(m entities.Message) sqsMessage {
	return sqsMessage{
		ID:        m.ID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
	}
}
