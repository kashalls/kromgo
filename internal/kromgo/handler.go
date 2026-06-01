// Package kromgo implements the HTTP handlers that turn Prometheus query results
// into JSON, SVG badges, sparkline charts, and raw/history responses.
package kromgo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
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

// handleInstant serves the json, raw, and badge formats. The value comes from an
// instant query, or a reduced range query when the metric's type is "range".
func (h *Handler) handleInstant(w http.ResponseWriter, r *http.Request, metric *resolvedMetric, format string, log *slog.Logger) {
	value, err := h.queryValue(r.Context(), metric)
	if err != nil {
		log.Error("error executing metric query", "error", err)
		writeError(w, metric.Name, "Query Error", http.StatusInternalServerError)
		return
	}

	if format == "raw" {
		if err := writeJSON(w, value); err != nil {
			log.Error("could not convert query result to json", "error", err)
			writeError(w, metric.Name, "Query Error", http.StatusInternalServerError)
		}
		return
	}

	vector, ok := value.(model.Vector)
	if !ok {
		log.Error("query did not return an instant vector", "type", value.Type().String())
		writeError(w, metric.Name, "Unexpected result type", http.StatusInternalServerError)
		return
	}

	if len(vector) == 0 {
		// No data: report it without evaluating the expressions (no result/labels).
		if err := writeJSON(w, EndpointResponse{SchemaVersion: 1, Label: metricTitle(metric), Message: "metric returned no data"}); err != nil {
			log.Error("error writing no-data response", "error", err)
			writeError(w, metric.Name, "Error", http.StatusInternalServerError)
		}
		return
	}

	message, color, ok := h.evalDisplay(metric, vector[0], log)
	if !ok {
		writeError(w, metric.Name, "Expression Error", http.StatusInternalServerError)
		return
	}
	title := metricTitle(metric)

	if format == "badge" {
		h.badges.write(w, r.URL.Query().Get("style"), title, message, color)
		return
	}

	resp := EndpointResponse{
		SchemaVersion: 1,
		Label:         title,
		Message:       message,
		Color:         color,               // omitted when empty
		CacheSeconds:  metric.cacheSeconds, // shields.io honors this; omitted when 0
	}
	if err := writeJSON(w, resp); err != nil {
		log.Error("error converting data to json response", "error", err)
		writeError(w, metric.Name, "Error", http.StatusInternalServerError)
	}
}

// evalDisplay evaluates the metric's value and color CEL expressions against a
// sample. ok is false only if the value expression errors (caller returns 500);
// a failing color expression is logged and treated as no color.
func (h *Handler) evalDisplay(metric *resolvedMetric, sample *model.Sample, log *slog.Logger) (message, color string, ok bool) {
	result := float64(sample.Value)
	labels := labelMap(sample.Metric)

	message, err := evalStringExpr(metric.valueProg, result, labels)
	if err != nil {
		log.Error("value expression failed", "error", err)
		return "", "", false
	}
	if metric.colorProg != nil {
		if color, err = evalStringExpr(metric.colorProg, result, labels); err != nil {
			log.Error("color expression failed", "error", err) // degrade to no color
			color = ""
		}
	}
	return message, color, true
}

// queryValue computes the metric's instant value: an instant query for the default
// type, or a range query reduced to one value per series for type: range.
func (h *Handler) queryValue(ctx context.Context, metric *resolvedMetric) (model.Value, error) {
	rq := metric.rangeQuery
	if rq == nil {
		return h.prom.Query(ctx, metric.Query, time.Now())
	}

	end := time.Now().Add(-rq.offset)
	value, err := h.prom.QueryRange(ctx, metric.Query, v1.Range{Start: end.Add(-rq.last), End: end, Step: rq.step})
	if err != nil {
		return nil, err
	}
	matrix, ok := value.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("range query returned %s, want matrix", value.Type())
	}
	return reduceMatrix(matrix, rq.reduce), nil
}

// labelMap converts a Prometheus label set to a plain string map for CEL/JSON use.
func labelMap(m model.Metric) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[string(k)] = string(v)
	}
	return out
}

// metricTitle returns the display title for a metric (its Title, falling back to Name).
func metricTitle(metric *resolvedMetric) string {
	if metric.Title != "" {
		return metric.Title
	}
	return metric.Name
}

func writeJSON(w http.ResponseWriter, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
	return nil
}

func writeSVG(w http.ResponseWriter, svg []byte) {
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(svg)
}

func requestLogger(r *http.Request, metric, format string) *slog.Logger {
	return slog.With(
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("metric", metric),
		slog.String("format", format),
	)
}
