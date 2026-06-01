package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := writeConfig(t, `
prometheus: http://prom:9090
defaults:
  hidden: false
  graph:
    maxDuration: 7d
badges:
  - id: cpu
    query: node_cpu
    value: string(result) + "%"
graphs:
  - id: cpu
    query: node_cpu
    width: 400
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "http://prom:9090", cfg.Prometheus)
	require.Len(t, cfg.Badges, 1)
	assert.Equal(t, "cpu", cfg.Badges[0].ID)
	require.Len(t, cfg.Graphs, 1)
	assert.Equal(t, 400, cfg.Graphs[0].Width)
	assert.Equal(t, "7d", cfg.Defaults.Graph.MaxDuration)
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	assert.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := Load(writeConfig(t, "badges: [: bad"))
	assert.Error(t, err)
}

func TestLoad_RejectsUnknownKey(t *testing.T) {
	// A non-legacy typo'd key must error under strict decoding.
	_, err := Load(writeConfig(t, "badges: []\nbogus: true\n"))
	assert.Error(t, err)
}

func TestLoad_LegacyConfigErrors(t *testing.T) {
	// A pre-0.12 config (top-level metrics:) gets a pointed migration error.
	_, err := Load(writeConfig(t, "metrics:\n  - name: cpu\n    query: q\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0.12")
}

func TestLoad_InvalidDuration(t *testing.T) {
	_, err := Load(writeConfig(t, "defaults:\n  graph:\n    maxDuration: bogus\n"))
	assert.Error(t, err)
}

func TestLoad_DuplicateID(t *testing.T) {
	_, err := Load(writeConfig(t, "badges:\n  - id: cpu\n    query: q\n  - id: cpu\n    query: q\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestLoad_RangeBadgeRequiresLast(t *testing.T) {
	_, err := Load(writeConfig(t, "badges:\n  - id: cpu\n    query: q\n    type: range\n"))
	assert.Error(t, err)
}

func TestLoadServer_Defaults(t *testing.T) {
	// Clear any inherited env so envDefault applies. t.Setenv registers the
	// restore; os.Unsetenv then removes the var for the duration of the test.
	for _, k := range []string{"SERVER_PORT", "HEALTH_PORT", "QUERY_TIMEOUT", "SERVER_LOGGING"} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	sc, err := LoadServer()
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", sc.ServerHost)
	assert.Equal(t, 8080, sc.ServerPort)
	assert.Equal(t, 8888, sc.HealthPort)
	assert.Equal(t, 30*time.Second, sc.QueryTimeout)
	assert.False(t, sc.ServerLogging)
}

func TestLoadServer_Overrides(t *testing.T) {
	t.Setenv("SERVER_PORT", "9000")
	t.Setenv("QUERY_TIMEOUT", "5s")
	t.Setenv("SERVER_LOGGING", "true")
	sc, err := LoadServer()
	require.NoError(t, err)
	assert.Equal(t, 9000, sc.ServerPort)
	assert.Equal(t, 5*time.Second, sc.QueryTimeout)
	assert.True(t, sc.ServerLogging)
}
