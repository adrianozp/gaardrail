package kafka

import (
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
)

type kafkaMessage struct {
	ID        string    `json:"id"`
	Body      []byte    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (k kafkaMessage) ToMessage() entities.Message {
	return entities.Message{
		ID:        k.ID,
		Body:      k.Body,
		CreatedAt: k.CreatedAt,
	}
}

func ToModel(m entities.Message) kafkaMessage {
	return kafkaMessage{
		ID:        m.ID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
	}
}
