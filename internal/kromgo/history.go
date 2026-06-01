package kromgo

import (
	"log/slog"
	"net/http"
)

type HistoryDataPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

type HistorySeries struct {
	Labels map[string]string  `json:"labels"`
	Data   []HistoryDataPoint `json:"data"`
}

type HistoryResponse struct {
	Metric string          `json:"metric"`
	Title  string          `json:"title"`
	Start  int64           `json:"start"`
	End    int64           `json:"end"`
	Step   int64           `json:"step"`
	Series []HistorySeries `json:"series"`
}

func (h *Handler) handleHistory(w http.ResponseWriter, r *http.Request, metric *resolvedMetric, log *slog.Logger) {
	start, end, step, ok := h.validateHistoryAccess(w, r, metric)
	if !ok {
		return
	}

	matrix, ok := h.queryMatrix(w, r, metric, start, end, step, log)
	if !ok {
		return
	}

	series := make([]HistorySeries, 0, len(matrix))
	for _, stream := range matrix {
		labels := labelMap(stream.Metric)
		data := make([]HistoryDataPoint, 0, len(stream.Values))
		for _, point := range stream.Values {
			data = append(data, HistoryDataPoint{
				T: int64(point.Timestamp) / 1000,
				V: float64(point.Value),
			})
		}
		series = append(series, HistorySeries{Labels: labels, Data: data})
	}

	resp := HistoryResponse{
		Metric: metric.Name,
		Title:  metricTitle(metric),
		Start:  start.Unix(),
		End:    end.Unix(),
		Step:   int64(step.Seconds()),
		Series: series,
	}

	if err := writeJSON(w, resp); err != nil {
		log.Error("error marshaling history response", "error", err)
		writeError(w, metric.Name, "Error", http.StatusInternalServerError)
	}
}
