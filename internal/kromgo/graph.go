package kromgo

import (
	"net/http"
	"time"

	"github.com/prometheus/common/model"
)

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

// serveGraph renders a time series as an SVG sparkline (default) or as JSON
// (?format=json).
func (h *Handler) serveGraph(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	format := r.URL.Query().Get("format")
	switch format {
	case formatJSON, formatPNG: // recognized
	default:
		format = formatSVG // empty or unknown renders the SVG image
	}

	metricLabel := "unknown"
	defer func() { requestsTotal.WithLabelValues("graph", metricLabel, format).Inc() }()
	log := requestLogger(r, "graph", id, format)

	graph, ok := h.graphs[id]
	if !ok {
		log.Error("graph not found")
		writeError(w, id, "Not Found", http.StatusNotFound)
		return
	}
	metricLabel = id
	setCache(w, graph.cacheSeconds)

	start, end, step, ok := h.validateGraphAccess(w, r, graph)
	if !ok {
		return
	}
	matrix, ok := h.queryMatrix(w, r, graph, start, end, step, log)
	if !ok {
		return
	}

	if format == formatJSON {
		writeJSONOr(w, log, id, historyResponse(graph, start, end, step, matrix))
		return
	}

	params := graph.defaults.withOverrides(r)
	img, err := renderChart(matrix, params)
	if err != nil {
		log.Error("error rendering chart", "error", err)
		writeError(w, id, "Render Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", params.contentType())
	_, _ = w.Write(img)
}

// historyResponse builds the JSON time-series payload from a query matrix.
func historyResponse(graph *resolvedGraph, start, end time.Time, step time.Duration, matrix model.Matrix) HistoryResponse {
	series := make([]HistorySeries, 0, len(matrix))
	for _, stream := range matrix {
		data := make([]HistoryDataPoint, 0, len(stream.Values))
		for _, point := range stream.Values {
			data = append(data, HistoryDataPoint{T: int64(point.Timestamp) / 1000, V: float64(point.Value)})
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
