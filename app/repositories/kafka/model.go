package kafka

import "github.com/adrianozp/gaardrail/app/entities"

type kafkaMessage struct {
	ID   string `json:"id"`
	Body []byte `json:"body"`
}

func (k kafkaMessage) ToMessage() entities.Message {
	return entities.Message{
		ID:   k.ID,
		Body: k.Body,
	}
}

func ToModel(m entities.Message) kafkaMessage {
	return kafkaMessage{
		ID:   m.ID,
		Body: m.Body,
	}
}
