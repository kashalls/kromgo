//go:build e2e

// Package e2e drives a compiled kromgo binary against a mock Prometheus.
package e2e

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/promtest"
	"github.com/stretchr/testify/require"
)

const configYAML = `
defaults:
  hidden: false
  graph:
    maxDuration: 24h
badges:
  - id: cpu
    query: node_cpu_usage
    value: string(result) + "%"
    color: 'result <= 50.0 ? "green" : "red"'
    icon: mdi:cpu-64-bit
graphs:
  - id: cpu
    query: node_cpu_usage
`

type harness struct {
	t         *testing.T
	baseURL   string
	healthURL string
	prom      *httptest.Server
	cmd       *exec.Cmd
}

// start compiles kromgo, launches it against a mock Prometheus, and waits for readiness.
func start(t *testing.T) *harness {
	t.Helper()

	root := moduleRoot(t)
	bin := filepath.Join(t.TempDir(), "kromgo")
	build := exec.Command("go", "build", "-o", bin, "./cmd/kromgo")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building kromgo: %v\n%s", err, out)
	}

	prom := promtest.Server(t, promtest.Scalar("17.5", map[string]string{"job": "node"}), []float64{10, 20, 15})

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o600))

	serverPort := freePort(t)
	healthPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, "-config", configPath)
	cmd.Env = append(os.Environ(),
		"PROMETHEUS_URL="+prom.URL,
		fmt.Sprintf("SERVER_PORT=%d", serverPort),
		fmt.Sprintf("HEALTH_PORT=%d", healthPort),
		"LOG_FORMAT=text",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())

	h := &harness{
		t:         t,
		baseURL:   fmt.Sprintf("http://127.0.0.1:%d", serverPort),
		healthURL: fmt.Sprintf("http://127.0.0.1:%d", healthPort),
		prom:      prom,
		cmd:       cmd,
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
	})

	h.waitReady()
	return h
}

func (h *harness) waitReady() {
	h.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(h.healthURL + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	h.t.Fatal("kromgo did not become ready within timeout")
}

// get performs a GET against the main server and returns the response.
func (h *harness) get(path string) *http.Response {
	h.t.Helper()
	resp, err := http.Get(h.baseURL + path)
	require.NoError(h.t, err)
	return resp
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

// moduleRoot walks up from this file's directory to the directory containing go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod")
		}
		dir = parent
	}
}
