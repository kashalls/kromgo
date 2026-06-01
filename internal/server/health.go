package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// healthMux serves Prometheus metrics and liveness/readiness probes.
func healthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	for _, path := range []string{"/healthz", "/-/health", "/readyz", "/-/ready"} {
		mux.HandleFunc("GET "+path, ok)
	}
	return mux
}

func ok(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
