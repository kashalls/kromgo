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

const shutdownTimeout = 30 * time.Second

// Run starts the application and health servers and blocks until ctx is cancelled
// or a server fails, then shuts both down gracefully.
func Run(ctx context.Context, sc config.ServerConfig, app http.Handler) error {
	main := &http.Server{
		Addr:         net.JoinHostPort(sc.ServerHost, strconv.Itoa(sc.ServerPort)),
		Handler:      withMiddleware(app, sc),
		ReadTimeout:  sc.ServerReadTimeout,
		WriteTimeout: sc.ServerWriteTimeout,
	}
	health := &http.Server{
		Addr:         net.JoinHostPort(sc.HealthHost, strconv.Itoa(sc.HealthPort)),
		Handler:      recoverer(healthMux()),
		ReadTimeout:  sc.ServerReadTimeout,
		WriteTimeout: sc.ServerWriteTimeout,
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
