package staticqueue_test

import (
	"testing"

	"github.com/adrianozp/gaardrail/app/repositories/queue/staticqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestType_ReturnsConfiguredProtocol(t *testing.T) {
	q := staticqueue.New("kafka")
	assert.Equal(t, "kafka", q.Type())
}

func TestAvailable_IsEmpty(t *testing.T) {
	q := staticqueue.New("kafka")
	assert.Empty(t, q.Available())
}

func TestSetType_AlwaysErrors(t *testing.T) {
	q := staticqueue.New("kafka")
	require.Error(t, q.SetType("inmemory"))
}
