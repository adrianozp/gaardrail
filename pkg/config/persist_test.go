package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const sampleYAML = `queue:
  protocol: kafka # boot queue
  capacity: 1000

controller:
  type: pid

pid:
  setpoint: 50.0
  kp: 0.005
`

func writeSample(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(sampleYAML), 0644))
	return path
}

func reload(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, yaml.Unmarshal(data, &out))
	return out
}

func TestSet_UpdatesExistingScalars(t *testing.T) {
	path := writeSample(t)
	p := NewPersister(Config{Path: path})

	require.NoError(t, p.Set(map[string]any{
		"queue.protocol":  "inmemory",
		"controller.type": "step",
		"pid.setpoint":    72.0,
	}))

	got := reload(t, path)
	assert.Equal(t, "inmemory", got["queue"].(map[string]any)["protocol"])
	assert.Equal(t, "step", got["controller"].(map[string]any)["type"])
	assert.EqualValues(t, 72.0, got["pid"].(map[string]any)["setpoint"])
}

func TestSet_PreservesCommentsAndUntouchedKeys(t *testing.T) {
	path := writeSample(t)
	p := NewPersister(Config{Path: path})

	require.NoError(t, p.Set(map[string]any{"queue.protocol": "constant"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# boot queue", "inline comments must survive")

	got := reload(t, path)
	assert.EqualValues(t, 1000, got["queue"].(map[string]any)["capacity"], "untouched keys must remain")
	assert.EqualValues(t, 0.005, got["pid"].(map[string]any)["kp"])
}

func TestSet_CreatesMissingKey(t *testing.T) {
	path := writeSample(t)
	p := NewPersister(Config{Path: path})

	require.NoError(t, p.Set(map[string]any{"queue.query": "SELECT 1: FROM t"}))

	got := reload(t, path)
	assert.Equal(t, "SELECT 1: FROM t", got["queue"].(map[string]any)["query"],
		"value with a colon must round-trip safely")
}

func TestSet_EmptyPathErrors(t *testing.T) {
	p := NewPersister(Config{Path: ""})
	require.Error(t, p.Set(map[string]any{"queue.protocol": "inmemory"}))
}
