package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/home-operations/kromgo/internal/config"
)

// withMiddleware wraps h with recovery, optional access logging, and optional rate limiting.
// Middleware is applied outermost-first: recover → access log → rate limit → handler.
func withMiddleware(h http.Handler, sc config.ServerConfig) http.Handler {
	if sc.RatelimitEnable {
		h = rateLimiter(sc)(h)
	}
	if sc.ServerLogging {
		h = accessLog(h)
	}
	return recoverer(h)
}

func rateLimiter(sc config.ServerConfig) func(http.Handler) http.Handler {
	switch {
	case sc.RatelimitAll:
		return httprate.LimitAll(sc.RatelimitRequestLimit, sc.RatelimitWindowLength)
	case sc.RatelimitByRealIP:
		return httprate.LimitByRealIP(sc.RatelimitRequestLimit, sc.RatelimitWindowLength)
	default:
		return httprate.LimitByIP(sc.RatelimitRequestLimit, sc.RatelimitWindowLength)
	}
}

// statusRecorder captures the response status code for access logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).String(),
		)
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(r.Context(), "panic recovered", "panic", rec, "path", r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
