package kafka

import (
	"encoding/json"

	"github.com/adrianozp/gaardrail/app/entities"
	kafkaclient "github.com/adrianozp/gaardrail/internal/kafka"
	"github.com/rs/zerolog/log"
)

type KafkaMessage struct {
	ID string `json:"id"`
}

type KafkaRepository struct {
	client *kafkaclient.Client
}

func NewKafkaRepository(client *kafkaclient.Client) *KafkaRepository {
	return &KafkaRepository{client: client}
}

func (r *KafkaRepository) Enqueue(m entities.Message) (string, error) {
	body, err := json.Marshal(ToModel(m))
	if err != nil {
		return "", err
	}
	if err := r.client.Publish(m.ID, body); err != nil {
		return "", err
	}
	return m.ID, nil
}

func (r *KafkaRepository) Dequeue() (entities.Message, error) {
	log.Debug().Msg("kafka: dequeing msg")
	msg := <-r.client.Messages()
	log.Debug().Msg("kafka: msg dequeued")

	var model kafkaMessage
	err := json.Unmarshal(msg.Value, &model)
	if err != nil {
		return entities.Message{}, err
	}

	entity := model.ToMessage()
	entity.Ack = func() error {
		r.client.MarkOffset(msg)
		return nil
	}
	return entity, nil
}

func (r *KafkaRepository) Ack(m entities.Message) error {
	if m.Ack != nil {
		return m.Ack()
	}
	return nil
}

func (r *KafkaRepository) Size() (int64, error) {
	return r.client.Size()
}
