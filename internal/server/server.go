// Package server wires kromgo's HTTP servers: the application server that serves
// metric endpoints and a health server that serves probes and Prometheus metrics.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/home-operations/kromgo/internal/config"
)

const (
	shutdownTimeout = 30 * time.Second
	// readHeaderTimeout bounds how long a client may take to send request headers,
	// independent of the (operator-tunable) body/response timeouts. Without it a
	// public listener is open to Slowloris-style header-drip attacks.
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 120 * time.Second
)

// Run starts the application and health servers and blocks until ctx is cancelled
// or a server fails, then shuts both down gracefully.
func Run(ctx context.Context, sc config.ServerConfig, app http.Handler) error {
	main := &http.Server{
		Addr:              net.JoinHostPort(sc.ServerHost, strconv.Itoa(sc.ServerPort)),
		Handler:           withMiddleware(app, sc),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       sc.ServerReadTimeout,
		WriteTimeout:      sc.ServerWriteTimeout,
		IdleTimeout:       idleTimeout,
	}
	health := &http.Server{
		Addr:              net.JoinHostPort(sc.HealthHost, strconv.Itoa(sc.HealthPort)),
		Handler:           recoverer(secureHeaders(healthMux())),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       sc.ServerReadTimeout,
		WriteTimeout:      sc.ServerWriteTimeout,
		IdleTimeout:       idleTimeout,
	}
	servers := []*http.Server{main, health}

	errCh := make(chan error, len(servers))
	for _, s := range servers {
		go func(s *http.Server) {
			slog.Info("server listening", "addr", s.Addr)
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("server %s: %w", s.Addr, err)
			}
		}(s)
	}

	select {
	case err := <-errCh:
		return errors.Join(err, shutdown(servers))
	case <-ctx.Done():
		slog.Info("shutdown signal received")
		return shutdown(servers)
	}
}

func shutdown(servers []*http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var errs []error
	for _, s := range servers {
		if err := s.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutting down %s: %w", s.Addr, err))
		}
	}
	return errors.Join(errs...)
}
