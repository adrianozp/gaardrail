package createmessage_test

import (
	"errors"
	"testing"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage/mocks"
	"github.com/adrianozp/gaardrail/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreate_Success(t *testing.T) {
	testUUID := "success-uuid"
	uuid.WithString(testUUID)
	msg := entities.Message{ID: testUUID, Body: []byte("msg-1")}
	mockQueue := mocks.NewQueue(t)
	mockQueue.On("Enqueue", msg).Return(testUUID, nil)

	uc := createmessage.NewCreateMessageUseCase(mockQueue)
	id, err := uc.Create(msg)

	assert.NoError(t, err)
	assert.Equal(t, testUUID, id)
}

func TestCreate_QueueError(t *testing.T) {
	testUUID := "queue-error-uuid"
	uuid.WithString(testUUID)
	msg := entities.Message{ID: testUUID, Body: []byte("msg-2")}
	mockQueue := mocks.NewQueue(t)
	mockQueue.On("Enqueue", msg).Return("", errors.New("queue unavailable"))

	uc := createmessage.NewCreateMessageUseCase(mockQueue)
	id, err := uc.Create(msg)

	assert.EqualError(t, err, "queue unavailable")
	assert.Empty(t, id)
}
