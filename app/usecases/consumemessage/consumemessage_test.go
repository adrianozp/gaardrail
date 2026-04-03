package consumemessage_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage"
	"github.com/adrianozp/gaardrail/app/usecases/consumemessage/mocks"
	"github.com/stretchr/testify/assert"
)

func TestConsume_Success(t *testing.T) {
	msg := entities.Message{ID: "msg-1"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue").Return(msg, nil)
	mockTarget.On("Push", msg).Return(nil)
	mockQueue.On("Ack", msg).Return(nil)

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume()

	assert.NoError(t, err)
	assert.Equal(t, "msg-1", id)
}

func TestConsume_DequeueError(t *testing.T) {
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue").Return(entities.Message{}, errors.New("dequeue failed"))

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume()

	assert.EqualError(t, err, "dequeue failed")
	assert.Empty(t, id)
}

func TestConsume_PushError(t *testing.T) {
	msg := entities.Message{ID: "msg-2"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue").Return(msg, nil)
	mockTarget.On("Push", msg).Return(errors.New("push failed"))

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume()

	assert.EqualError(t, err, "push failed")
	assert.Empty(t, id)
}

func TestConsume_AckError(t *testing.T) {
	msg := entities.Message{ID: "msg-3"}
	mockQueue := mocks.NewQueue(t)
	mockTarget := mocks.NewTarget(t)
	mockQueue.On("Dequeue").Return(msg, nil)
	mockTarget.On("Push", msg).Return(nil)
	mockQueue.On("Ack", msg).Return(errors.New("ack failed"))

	uc := consumemessage.NewConsumeMessageUseCase(mockQueue, mockTarget)
	id, err := uc.Consume()

	assert.EqualError(t, err, "ack failed")
	assert.Empty(t, id)
}
