package kafka

import "github.com/adrianozp/gaardrail/app/entities"

type kafkaMessage struct {
	ID    string
	Topic string
}

func (k kafkaMessage) ToMessage() entities.Message {
	return entities.Message{
		ID: k.ID,
	}
}

func ToModel(m entities.Message) kafkaMessage {
	return kafkaMessage{
		ID: m.ID,
	}
}
