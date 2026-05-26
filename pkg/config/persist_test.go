package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	viper.Reset()
	viper.SetConfigFile(path)
	require.NoError(t, viper.ReadInConfig())
	return path
}

func TestPersistControllerType_AddsSectionPreservingRest(t *testing.T) {
	path := writeTempConfig(t, "http:\n  addr: \":8080\"  # bind addr\nqueue:\n  protocol: kafka\n")

	require.NoError(t, ControllerTypeStore{}.PersistControllerType("step"))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "controller:")
	assert.Contains(t, s, "type: step")
	assert.Contains(t, s, "# bind addr", "inline comments must be preserved")
	assert.Contains(t, s, "protocol: kafka", "untouched keys must remain")
}

func TestPersistControllerType_UpdatesExistingInPlace(t *testing.T) {
	path := writeTempConfig(t, "controller:\n  type: pid\nqueue:\n  protocol: kafka\n")

	require.NoError(t, ControllerTypeStore{}.PersistControllerType("step"))

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(out), "type: step")
	assert.NotContains(t, string(out), "type: pid")
}

func TestPersistControllerType_NoConfigFileErrors(t *testing.T) {
	viper.Reset() // no config file in use

	require.Error(t, ControllerTypeStore{}.PersistControllerType("step"))
}
