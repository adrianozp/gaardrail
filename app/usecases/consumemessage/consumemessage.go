package consumemessage

import (
	"context"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/rs/zerolog/log"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Queue interface {
	Dequeue(context.Context) (entities.Message, error)
	Ack(context.Context, entities.Message) error
	Size() (int64, error)
}

type Target interface {
	Push(context.Context, entities.Message) error
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

func (u ConsumeMessageUseCase) Consume(ctx context.Context) (string, error) {
	consumeStartTime := clock.Now()
	message, err := u.queue.Dequeue(ctx)
	metrics.Gauge(map[string]float64{"dequeue_time_seconds": clock.Now().Sub(consumeStartTime).Seconds()})
	if err != nil {
		log.Error().Msg("error dequeing message")
		return "", err
	}

	err = u.target.Push(ctx, message)
	if err != nil {
		log.Error().Str("message_id", message.ID).Msg("error pushing message")
		return "", err
	}

	err = u.queue.Ack(ctx, message)
	if err != nil {
		log.Error().Str("message_id", message.ID).Msg("error consuming message")
		return "", err
	}

	sendMetrics(consumeStartTime, message)

	return message.ID, nil
}

func sendMetrics(consumeStartTime time.Time, message entities.Message) {
	consumeTime := clock.Now().Sub(consumeStartTime).Seconds()
	processTime := clock.Now().Sub(message.CreatedAt).Seconds()
	log.Debug().Str("message_id", message.ID).Float64("consume_time", consumeTime).Float64("process_time", processTime).Msg("consumed message")
	metrics.Incr([]string{"messages_processed_total"})
	metrics.Gauge(map[string]float64{
		"consume_time_seconds": consumeTime,
		"process_time_seconds": processTime,
	})
}

func (u ConsumeMessageUseCase) Size() (int64, error) {
	return u.queue.Size()
}
