package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/logging"
)

// withMiddleware wraps h with request-logger injection, recovery, optional access
// logging, and security headers. Middleware is applied outermost-first: inject logger
// → recover → access log → security headers → handler. Rate limiting is intentionally
// left to a reverse proxy (see the README).
func withMiddleware(h http.Handler, sc config.ServerConfig) http.Handler {
	h = secureHeaders(h)
	if sc.ServerLogging {
		h = accessLog(h)
	}
	return injectLogger(recoverer(h))
}

var requestCounter atomic.Uint64

// injectLogger attaches a request-scoped logger (request_id, method, path) to the
// request context so downstream middleware and handlers share one set of base fields.
func injectLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := slog.With(
			slog.String("request_id", strconv.FormatUint(requestCounter.Add(1), 36)),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
	})
}

// secureHeaders sets defensive response headers. nosniff stops MIME confusion; the
// CSP neutralizes any markup that slips into an SVG (responses carry no scripts and
// only inline style attributes), so a metric label can't execute as script even if
// the SVG is opened as a top-level document.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
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
		logging.FromContext(r.Context()).LogAttrs(r.Context(), slog.LevelInfo, "request",
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logging.FromContext(r.Context()).LogAttrs(r.Context(), slog.LevelError, "panic recovered",
					slog.Any("panic", rec),
				)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
