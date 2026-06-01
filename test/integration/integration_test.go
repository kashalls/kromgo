//go:build integration

// Package integration exercises kromgo against a real Prometheus.
// It is skipped unless PROMETHEUS_URL is set, mirroring the org's env-gated
// integration pattern. Run with:
//
//	PROMETHEUS_URL=http://localhost:9090 go test -tags integration ./test/integration/...
package integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/kromgo"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHandler(t *testing.T) *kromgo.Handler {
	t.Helper()
	url := os.Getenv("PROMETHEUS_URL")
	if url == "" {
		t.Skip("PROMETHEUS_URL not set; skipping integration test")
	}

	client, err := prometheus.New(url, 30*time.Second)
	require.NoError(t, err)

	cfg := config.KromgoConfig{
		Metrics: []config.Metric{
			{Name: "up", Query: "sum(up)"},
		},
		Defaults: config.Defaults{Timeseries: config.TimeseriesConfig{Enabled: true, MaxDuration: "24h"}},
	}
	h, err := kromgo.New(cfg, client)
	require.NoError(t, err)
	return h
}

func get(t *testing.T, h *kromgo.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	h.Mux().ServeHTTP(w, req)
	return w
}

func TestIntegration_JSON(t *testing.T) {
	h := newHandler(t)

	w := get(t, h, "/up")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"label":"up"`)
}

func TestIntegration_History(t *testing.T) {
	h := newHandler(t)

	w := get(t, h, "/up?format=history&last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"metric":"up"`)
}
