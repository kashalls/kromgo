package kromgo

import (
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/home-operations/kromgo/internal/logging"
	"github.com/prometheus/common/model"
)

// maxGraphSeries caps how many series a single graph response will encode or render.
// The series count tracks the exposed metric's label cardinality, which is not
// request-bounded and can spike at runtime (label churn); without a cap one public
// request could materialize and draw thousands of lines. Excess series are dropped
// and the truncation is logged. It is a fixed limit, like maxChartDimension.
const maxGraphSeries = 100

// HistoryDataPoint is one sample in a graph's JSON time series.
type HistoryDataPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

// HistorySeries is one labelled series in a graph's JSON time series.
type HistorySeries struct {
	Labels map[string]string  `json:"labels"`
	Data   []HistoryDataPoint `json:"data"`
}

// HistoryResponse is the JSON returned for a graph's ?format=json.
type HistoryResponse struct {
	ID     string          `json:"id"`
	Title  string          `json:"title"`
	Start  int64           `json:"start"`
	End    int64           `json:"end"`
	Step   int64           `json:"step"`
	Series []HistorySeries `json:"series"`
}

// graphFormat resolves a graph request's output format, defaulting to SVG for an
// empty or unrecognized value.
func graphFormat(r *http.Request) string {
	switch f := r.URL.Query().Get("format"); f {
	case formatJSON, formatPNG:
		return f
	default:
		return formatSVG
	}
}

// serveGraph renders a time series as an SVG sparkline (default) or as JSON
// (?format=json).
func (h *Handler) serveGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	format := graphFormat(r)

	metricLabel := "unknown"
	defer func() { requestsTotal.WithLabelValues("graph", metricLabel, format).Inc() }()
	log := logging.FromContext(r.Context()).With("kind", "graph", "id", id, "format", format)

	graph, ok := h.graphs[id]
	if !ok {
		log.Error("graph not found")
		h.errorResponse(w, format, id, "Not Found", http.StatusNotFound)
		return
	}
	metricLabel = id
	h.cache.apply(w)

	start, end, step, ok := h.validateGraphAccess(w, r, graph)
	if !ok {
		return
	}
	matrix, ok := h.queryMatrix(w, r, graph, start, end, step, log)
	if !ok {
		return
	}
	matrix = capSeries(matrix, log)

	if format == formatJSON {
		writeJSONOr(w, log, id, historyResponse(graph, start, end, step, matrix))
		return
	}

	params := graph.defaults.withOverrides(r)
	img, err := renderChart(matrix, params)
	if err != nil {
		log.Error("error rendering chart", "error", err)
		h.errorResponse(w, format, id, "Render Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", params.contentType())
	_, _ = w.Write(img)
}

// capSeries truncates a matrix to maxGraphSeries, logging when it drops series so a
// silent truncation doesn't read as "this is the whole result".
func capSeries(matrix model.Matrix, log *slog.Logger) model.Matrix {
	if len(matrix) <= maxGraphSeries {
		return matrix
	}
	log.Warn("graph series truncated", "total", len(matrix), "cap", maxGraphSeries)
	return matrix[:maxGraphSeries]
}

// historyResponse builds the JSON time-series payload from a query matrix.
func historyResponse(graph *resolvedGraph, start, end time.Time, step time.Duration, matrix model.Matrix) HistoryResponse {
	series := make([]HistorySeries, 0, len(matrix))
	for _, stream := range matrix {
		data := make([]HistoryDataPoint, 0, len(stream.Values))
		for _, point := range stream.Values {
			v := float64(point.Value)
			// Skip non-finite samples: encoding/json errors on NaN/Inf (a single such
			// sample would 500 the whole response), and the chart renders them as gaps.
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			data = append(data, HistoryDataPoint{T: int64(point.Timestamp) / 1000, V: v})
		}
		series = append(series, HistorySeries{Labels: labelMap(stream.Metric), Data: data})
	}
	return HistoryResponse{
		ID:     graph.ID,
		Title:  displayTitle(graph.Title, graph.ID),
		Start:  start.Unix(),
		End:    end.Unix(),
		Step:   int64(step.Seconds()),
		Series: series,
	}
}
