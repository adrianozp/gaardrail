package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("APP_HTTP_ADDR", ":9999")
	t.Setenv("APP_KAFKA_TOPIC", "test-topic")
	t.Setenv("APP_PID_SETPOINT", "42.0")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, ":9999", cfg.HTTP.Addr)
	assert.Equal(t, "test-topic", cfg.Kafka.Topic)
	assert.Equal(t, 42.0, cfg.PID.Setpoint)
}
