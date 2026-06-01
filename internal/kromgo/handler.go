// Package kromgo implements the HTTP handlers that turn Prometheus query results
// into JSON, SVG badges, sparkline charts, and raw/history responses.
package kromgo

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/prometheus/common/model"
)

// Handler serves metric endpoints backed by Prometheus queries.
type Handler struct {
	cfg     config.KromgoConfig
	metrics map[string]config.Metric
	prom    *prometheus.Client
	badges  *badgePool
}

// New builds a Handler from config and a Prometheus client.
func New(cfg config.KromgoConfig, prom *prometheus.Client) (*Handler, error) {
	badges, err := newBadgePool(cfg.Badge)
	if err != nil {
		return nil, err
	}
	return &Handler{
		cfg:     cfg,
		metrics: cfg.MetricsByName(),
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
	if format == "" {
		format = "json"
	}

	defer requestsTotal.WithLabelValues(name, format).Inc()
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

	switch format {
	case "history":
		h.handleHistory(w, r, metric, log)
	case "chart":
		h.handleChart(w, r, metric, log)
	default:
		h.handleInstant(w, r, metric, format, log)
	}
}

// handleInstant serves the json, raw, and badge formats, which all run an instant query.
func (h *Handler) handleInstant(w http.ResponseWriter, r *http.Request, metric config.Metric, format string, log *slog.Logger) {
	value, err := h.prom.Query(r.Context(), metric.Query, time.Now())
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

	var colorConfig config.MetricColor
	var response string
	if len(vector) > 0 {
		resultValue := float64(vector[0].Value)
		colorConfig = GetColorConfig(metric.Colors, resultValue)
		response = strconv.FormatFloat(resultValue, 'f', -1, 64)
	} else {
		response = "metric returned no data"
	}

	if metric.Label != "" {
		labelValue, err := ExtractLabelValue(vector, metric.Label)
		if err != nil {
			log.Error("label was not found in query result", "label", metric.Label, "error", err)
			writeError(w, metric.Name, "No Data", http.StatusOK)
			return
		}
		response = labelValue
	}
	if colorConfig.ValueOverride != "" {
		response = colorConfig.ValueOverride
	}

	if metric.ValueTemplate != "" {
		tmplStr := metric.ValueTemplate
		if resolved, ok := h.cfg.Templates[tmplStr]; ok {
			tmplStr = resolved
		}
		formatted, err := ApplyValueTemplate(tmplStr, response)
		if err != nil {
			log.Error("failed to apply value template", "error", err)
		}
		response = formatted
	}

	message := metric.Prefix + response + metric.Suffix
	title := metricTitle(metric)

	if format == "badge" {
		h.badges.write(w, r.URL.Query().Get("style"), title, message, colorConfig.Color)
		return
	}

	data := map[string]any{
		"schemaVersion": 1,
		"label":         title,
		"message":       message,
	}
	if colorConfig.Color != "" {
		data["color"] = colorConfig.Color
	}
	if err := writeJSON(w, data); err != nil {
		log.Error("error converting data to json response", "error", err)
		writeError(w, metric.Name, "Error", http.StatusInternalServerError)
	}
}

// metricTitle returns the display title for a metric (its Title, falling back to Name).
func metricTitle(metric config.Metric) string {
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
		"req_method", r.Method,
		"req_path", r.URL.Path,
		"metric", metric,
		"format", format,
	)
}
