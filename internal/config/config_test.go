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
metrics:
  - name: cpu
    query: node_cpu
    suffix: "%"
defaults:
  timeseries:
    enabled: true
    maxDuration: 7d
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "http://prom:9090", cfg.Prometheus)
	require.Len(t, cfg.Metrics, 1)
	assert.Equal(t, "cpu", cfg.Metrics[0].Name)
	assert.True(t, cfg.Defaults.Timeseries.Enabled)
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	assert.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := Load(writeConfig(t, "metrics: [: bad"))
	assert.Error(t, err)
}

func TestLoad_InvalidDuration(t *testing.T) {
	_, err := Load(writeConfig(t, "defaults:\n  timeseries:\n    maxDuration: bogus\n"))
	assert.Error(t, err)
}

func TestMetricsByName(t *testing.T) {
	cfg := KromgoConfig{Metrics: []Metric{{Name: "a"}, {Name: "b"}}}
	idx := cfg.MetricsByName()
	assert.Len(t, idx, 2)
	assert.Equal(t, "a", idx["a"].Name)
	_, ok := idx["missing"]
	assert.False(t, ok)
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
