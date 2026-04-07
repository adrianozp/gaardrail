package createmessage_test

import (
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage"
	"github.com/adrianozp/gaardrail/app/usecases/createmessage/mocks"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/clock"
	"github.com/adrianozp/gaardrail/pkg/uuid"
	"github.com/stretchr/testify/assert"
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

func TestCreate_RecordsEnqueueTime(t *testing.T) {
	t0 := time.Unix(0, 0)
	call := 0
	clock.With(func() time.Time {
		call++
		if call == 1 {
			return t0
		}
		return t0.Add(150 * time.Millisecond)
	})
	t.Cleanup(func() { clock.With(nil) })

	testUUID := "enqueue-time-uuid"
	uuid.WithString(testUUID)
	msg := entities.Message{ID: testUUID, Body: []byte("sql")}
	mockQueue := mocks.NewQueue(t)
	mockQueue.On("Enqueue", msg).Return(testUUID, nil)

	rec := &captureRecorder{gauged: map[string]float64{}}
	metrics.SetRecorder(rec)
	t.Cleanup(func() { metrics.SetRecorder(&captureRecorder{gauged: map[string]float64{}}) })

	uc := createmessage.NewCreateMessageUseCase(mockQueue)
	_, err := uc.Create(msg)

	require.NoError(t, err)
	assert.Equal(t, 0.15, rec.gauged["enqueue_time_seconds"])
}
