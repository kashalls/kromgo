// Package kromgo implements the HTTP handlers that turn Prometheus query results
// into JSON, SVG badges, sparkline charts, and raw/history responses.
package kromgo

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/prometheus/common/model"
)

// Handler serves metric endpoints backed by Prometheus queries.
type Handler struct {
	cfg     config.KromgoConfig
	metrics map[string]*resolvedMetric
	prom    *prometheus.Client
	badges  *badgePool
}

// New builds a Handler from config and a Prometheus client. Per-metric CEL
// expressions and durations are compiled/parsed here, so malformed config fails
// at startup rather than on a request.
func New(cfg config.KromgoConfig, prom *prometheus.Client) (*Handler, error) {
	badges, err := newBadgePool(cfg.Badge)
	if err != nil {
		return nil, err
	}

	env, err := newCELEnv()
	if err != nil {
		return nil, fmt.Errorf("building CEL environment: %w", err)
	}

	metrics := make(map[string]*resolvedMetric, len(cfg.Metrics))
	for _, m := range cfg.Metrics {
		rm, err := resolveMetric(m, cfg, env)
		if err != nil {
			return nil, err
		}
		metrics[m.Name] = rm
	}

	return &Handler{
		cfg:     cfg,
		metrics: metrics,
		prom:    prom,
		badges:  badges,
	}, nil
}

// Mux returns the application router: the index page and per-metric endpoints.
func (h *Handler) Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /{metric}", h.serveMetric)
	return mux
}

// serveMetric dispatches a metric request to the handler for its requested format.
func (h *Handler) serveMetric(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("metric")
	if name == "query" {
		name = r.URL.Query().Get("metric")
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "raw", "badge", "chart", "history": // recognized formats
	default:
		format = "json" // empty or unknown falls through to JSON
	}

	// metricLabel is bounded to configured names (plus "unknown") rather than the
	// raw request path, so arbitrary URLs can't explode the counter's cardinality.
	metricLabel := "unknown"
	defer func() { requestsTotal.WithLabelValues(metricLabel, format).Inc() }()
	log := requestLogger(r, name, format)

	if name == "" {
		writeError(w, name, "A valid metric name must be passed: /{metric}", http.StatusBadRequest)
		return
	}

	metric, exists := h.metrics[name]
	if !exists {
		log.Error("metric not found")
		writeError(w, name, "Not Found", http.StatusNotFound)
		return
	}
	metricLabel = name

	// Set the cache policy up front; writeError overrides it with no-store on failures.
	if metric.cacheSeconds > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", metric.cacheSeconds))
	}

	switch format {
	case "history":
		h.handleHistory(w, r, metric, log)
	case "chart":
		h.handleChart(w, r, metric, log)
	default:
		h.handleInstant(w, r, metric, format, log)
	}
}

// labelMap converts a Prometheus label set to a plain string map for CEL/JSON use.
func labelMap(m model.Metric) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[string(k)] = string(v)
	}
	return out
}

func requestLogger(r *http.Request, metric, format string) *slog.Logger {
	return slog.With(
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("metric", metric),
		slog.String("format", format),
	)
}
