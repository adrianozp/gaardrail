package consumemessage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage/mocks"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type captureRecorder struct {
	gauged map[string]float64
}

func (r *captureRecorder) Gauge(vals map[string]float64) {
	for k, v := range vals {
		r.gauged[k] = v
	}
}
func (r *captureRecorder) Incr([]string) {}

func TestConsume_Success(t *testing.T) {
	ctx := context.Background()
	msg := entities.Message{ID: "msg-1"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue", mock.Anything).Return(msg, nil)
	mockTarget.On("Push", mock.Anything, msg).Return(nil)
	mockQueue.On("Ack", mock.Anything, msg).Return(nil)

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "msg-1", id)
}

func TestConsume_DequeueError(t *testing.T) {
	ctx := context.Background()
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue", mock.Anything).Return(entities.Message{}, errors.New("dequeue failed"))

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume(ctx)

	assert.EqualError(t, err, "dequeue failed")
	assert.Empty(t, id)
}

func TestConsume_PushError(t *testing.T) {
	ctx := context.Background()
	msg := entities.Message{ID: "msg-2"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue", mock.Anything).Return(msg, nil)
	mockTarget.On("Push", mock.Anything, msg).Return(errors.New("push failed"))

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume(ctx)

	assert.EqualError(t, err, "push failed")
	assert.Empty(t, id)
}

func TestConsume_AckError(t *testing.T) {
	ctx := context.Background()
	msg := entities.Message{ID: "msg-3"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue", mock.Anything).Return(msg, nil)
	mockTarget.On("Push", mock.Anything, msg).Return(nil)
	mockQueue.On("Ack", mock.Anything, msg).Return(errors.New("ack failed"))

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume(ctx)

	assert.EqualError(t, err, "ack failed")
	assert.Empty(t, id)
}

func TestConsume_RecordsDequeueTime(t *testing.T) {
	t0 := time.Unix(0, 0)
	call := 0
	clock.With(func() time.Time {
		call++
		switch call {
		case 1:
			return t0 // consumeStartTime
		case 2:
			return t0.Add(200 * time.Millisecond) // after Dequeue
		default:
			return t0.Add(250 * time.Millisecond) // sendMetrics calls
		}
	})
	t.Cleanup(func() { clock.With(nil) })

	ctx := context.Background()
	msg := entities.Message{ID: "msg-1"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue", mock.Anything).Return(msg, nil)
	mockTarget.On("Push", mock.Anything, msg).Return(nil)
	mockQueue.On("Ack", mock.Anything, msg).Return(nil)

	rec := &captureRecorder{gauged: map[string]float64{}}
	metrics.SetRecorder(rec)
	t.Cleanup(func() { metrics.SetRecorder(&captureRecorder{gauged: map[string]float64{}}) })

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	_, err := uc.Consume(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0.2, rec.gauged["dequeue_time_seconds"])
}
