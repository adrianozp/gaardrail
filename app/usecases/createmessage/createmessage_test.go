package createmessage_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCreate_Success(t *testing.T) {
	msg := entities.Message{ID: "msg-1"}
	mockQueue := mocks.NewQueue(t)
	mockQueue.On("Enqueue", msg).Return("generated-id", nil)

	uc := createmessage.NewCreateMessageUseCase(mockQueue)
	id, err := uc.Create(msg)

	assert.NoError(t, err)
	assert.Equal(t, "generated-id", id)
}

func TestCreate_QueueError(t *testing.T) {
	msg := entities.Message{ID: "msg-2"}
	mockQueue := mocks.NewQueue(t)
	mockQueue.On("Enqueue", msg).Return("", errors.New("queue unavailable"))

	uc := createmessage.NewCreateMessageUseCase(mockQueue)
	id, err := uc.Create(msg)

	assert.EqualError(t, err, "queue unavailable")
	assert.Empty(t, id)
}
