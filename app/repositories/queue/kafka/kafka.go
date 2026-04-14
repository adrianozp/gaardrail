package kafka

import (
	"context"
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

func (r *KafkaRepository) Dequeue(ctx context.Context) (entities.Message, error) {
	select {
	case <-ctx.Done():
		return entities.Message{}, ctx.Err()
	case msg := <-r.client.Messages():
		log.Debug().Msg("kafka: msg dequeued")

		var model kafkaMessage
		if err := json.Unmarshal(msg.Value, &model); err != nil {
			return entities.Message{}, err
		}

		entity := model.ToMessage()
		entity.Ack = func() error {
			r.client.MarkOffset(msg)
			return nil
		}
		return entity, nil
	}
}

func (r *KafkaRepository) Ack(_ context.Context, m entities.Message) error {
	if m.Ack != nil {
		return m.Ack()
	}
	return nil
}

func (r *KafkaRepository) Size() (int64, error) {
	return r.client.Size()
}
