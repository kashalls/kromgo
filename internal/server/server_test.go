package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureHeaders(t *testing.T) {
	t.Parallel()
	h := secureHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
}

func TestHealthMux(t *testing.T) {
	t.Parallel()
	mux := healthMux()
	for _, path := range []string{"/healthz", "/-/health", "/readyz", "/-/ready", "/metrics"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestRecoverer_TurnsPanicInto500(t *testing.T) {
	t.Parallel()
	h := recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAccessLog_PassesThrough(t *testing.T) {
	t.Parallel()
	h := accessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hi"))
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusTeapot, w.Code)
	assert.Equal(t, "hi", w.Body.String())
}

func TestWithMiddleware_LoggingOptional(t *testing.T) {
	t.Parallel()
	called := false
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	for _, logging := range []bool{false, true} {
		called = false
		h := withMiddleware(base, config.ServerConfig{ServerLogging: logging})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	}
}

func TestRun_GracefulShutdown(t *testing.T) {
	t.Parallel()
	sc := config.ServerConfig{
		ServerHost: "127.0.0.1", ServerPort: testutil.FreePort(t),
		HealthHost: "127.0.0.1", HealthPort: testutil.FreePort(t),
	}
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, sc, app) }()

	// Wait until the health server is serving, then trigger graceful shutdown.
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", sc.HealthPort)
	require.Eventually(t, func() bool {
		resp, err := http.Get(healthURL)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*time.Second, 20*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}
