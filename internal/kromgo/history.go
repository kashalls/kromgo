package kromgo

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
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

var (
	errStartAfterEnd       = &historyParamError{"start must be before end"}
	errNonPositiveDuration = &historyParamError{"last must be a positive duration"}
)

type historyParamError struct{ msg string }

func (e *historyParamError) Error() string { return e.msg }

func parseTimeParam(s string) (time.Time, error) {
	// Try Unix timestamp (integer) first, then fall back to RFC3339.
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	return time.Parse(time.RFC3339, s)
}

func parseHistoryParams(r *http.Request) (start, end time.Time, step time.Duration, err error) {
	now := time.Now()
	q := r.URL.Query()

	// last=7d is shorthand for start=now-7d&end=now.
	if s := q.Get("last"); s != "" {
		d, derr := config.ParseDuration(s)
		if derr != nil {
			return start, end, step, derr
		}
		if d <= 0 {
			return start, end, step, errNonPositiveDuration
		}
		end, start = now, now.Add(-d)
	} else {
		end = now
		if s := q.Get("end"); s != "" {
			if end, err = parseTimeParam(s); err != nil {
				return start, end, step, err
			}
		}

		start = end.Add(-1 * time.Hour)
		if s := q.Get("start"); s != "" {
			if start, err = parseTimeParam(s); err != nil {
				return start, end, step, err
			}
		}

		if start.After(end) {
			return start, end, step, errStartAfterEnd
		}
	}

	// Auto step is 1/100th of the window, clamped to a 1m minimum.
	minStep := time.Minute
	step = max(end.Sub(start)/100, minStep)
	if s := q.Get("step"); s != "" {
		if step, err = config.ParseDuration(s); err != nil {
			return start, end, step, err
		}
		step = max(step, minStep)
	}

	return start, end, step, nil
}

func (h *Handler) historyEnabled(metric config.Metric) bool {
	if metric.History != nil && metric.History.Enabled != nil {
		return *metric.History.Enabled
	}
	return h.cfg.History.Enabled
}

func (h *Handler) historyMaxDuration(metric config.Metric) time.Duration {
	// Values are validated at startup by config.Load, so Parse cannot fail here.
	if metric.History != nil && metric.History.MaxDuration != "" {
		d, _ := config.ParseDuration(metric.History.MaxDuration)
		return d
	}
	if h.cfg.History.MaxDuration != "" {
		d, _ := config.ParseDuration(h.cfg.History.MaxDuration)
		return d
	}
	return time.Hour
}

// validateHistoryAccess checks access control, parses time parameters, and enforces the
// max duration cap. Returns ok=false if an error response was already written.
func (h *Handler) validateHistoryAccess(w http.ResponseWriter, r *http.Request, metric config.Metric) (start, end time.Time, step time.Duration, ok bool) {
	if !h.historyEnabled(metric) {
		writeError(w, metric.Name, "History not enabled for this metric", http.StatusForbidden)
		return start, end, step, false
	}

	start, end, step, err := parseHistoryParams(r)
	if err != nil {
		writeError(w, metric.Name, "Invalid parameter: "+err.Error(), http.StatusBadRequest)
		return start, end, step, false
	}

	if maxDur := h.historyMaxDuration(metric); maxDur > 0 && end.Sub(start) > maxDur {
		writeError(w, metric.Name, "Requested time window exceeds maximum allowed duration", http.StatusBadRequest)
		return start, end, step, false
	}

	return start, end, step, true
}

func (h *Handler) handleHistory(w http.ResponseWriter, r *http.Request, metric config.Metric, log *slog.Logger) {
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
		labels := make(map[string]string, len(stream.Metric))
		for k, v := range stream.Metric {
			labels[string(k)] = string(v)
		}
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

// queryMatrix runs a range query and asserts the result is a matrix, writing an error
// response and returning ok=false otherwise.
func (h *Handler) queryMatrix(w http.ResponseWriter, r *http.Request, metric config.Metric, start, end time.Time, step time.Duration, log *slog.Logger) (model.Matrix, bool) {
	value, err := h.prom.QueryRange(r.Context(), metric.Query, v1.Range{Start: start, End: end, Step: step})
	if err != nil {
		log.Error("error executing range query", "error", err)
		writeError(w, metric.Name, "Query Error", http.StatusInternalServerError)
		return nil, false
	}
	matrix, ok := value.(model.Matrix)
	if !ok {
		log.Error("range query did not return a matrix", "type", value.Type().String())
		writeError(w, metric.Name, "Unexpected result type", http.StatusInternalServerError)
		return nil, false
	}
	return matrix, true
}
