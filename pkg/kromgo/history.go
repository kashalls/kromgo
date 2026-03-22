package kromgo

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
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

func parseTimeParam(s string) (time.Time, error) {
	// Try Unix timestamp (integer)
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	// Try RFC3339
	return time.Parse(time.RFC3339, s)
}

func parseHistoryParams(r *http.Request) (start, end time.Time, step time.Duration, err error) {
	now := time.Now()

	// last=7d is shorthand for start=now-7d&end=now
	if s := r.URL.Query().Get("last"); s != "" {
		var d time.Duration
		d, err = configuration.ParseDuration(s)
		if err != nil {
			return
		}
		if d <= 0 {
			err = errNonPositiveDuration
			return
		}
		end = now
		start = now.Add(-d)
	} else {
		// Parse end
		end = now
		if s := r.URL.Query().Get("end"); s != "" {
			end, err = parseTimeParam(s)
			if err != nil {
				return
			}
		}

		// Parse start
		start = end.Add(-1 * time.Hour)
		if s := r.URL.Query().Get("start"); s != "" {
			start, err = parseTimeParam(s)
			if err != nil {
				return
			}
		}

		if start.After(end) {
			err = errStartAfterEnd
			return
		}
	}

	// Parse step
	minStep := time.Minute
	step = max(end.Sub(start)/100, minStep)
	if s := r.URL.Query().Get("step"); s != "" {
		step, err = configuration.ParseDuration(s)
		if err != nil {
			return
		}
		step = max(step, minStep)
	}

	return
}

var errStartAfterEnd = &historyParamError{"start must be before end"}
var errNonPositiveDuration = &historyParamError{"last must be a positive duration"}

type historyParamError struct{ msg string }

func (e *historyParamError) Error() string { return e.msg }

func (h *KromgoHandler) historyEnabled(metric configuration.Metric) bool {
	if metric.History != nil && metric.History.Enabled != nil {
		return *metric.History.Enabled
	}
	return h.Config.History.Enabled
}

func (h *KromgoHandler) historyMaxDuration(metric configuration.Metric) time.Duration {
	// Values are validated at startup by configuration.Init, so Parse cannot fail here.
	if metric.History != nil && metric.History.MaxDuration != "" {
		d, _ := configuration.ParseDuration(metric.History.MaxDuration)
		return d
	}
	if h.Config.History.MaxDuration != "" {
		d, _ := configuration.ParseDuration(h.Config.History.MaxDuration)
		return d
	}
	return time.Hour
}

// validateHistoryAccess checks access control, parses time parameters, and enforces
// the max duration cap. Returns ok=false if an error response was already written.
func (h *KromgoHandler) validateHistoryAccess(w http.ResponseWriter, r *http.Request, metric configuration.Metric) (start, end time.Time, step time.Duration, ok bool) {
	if !h.historyEnabled(metric) {
		HandleError(w, r, metric.Name, "History not enabled for this metric", http.StatusForbidden)
		return
	}

	var err error
	start, end, step, err = parseHistoryParams(r)
	if err != nil {
		HandleError(w, r, metric.Name, "Invalid parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	if maxDur := h.historyMaxDuration(metric); maxDur > 0 && end.Sub(start) > maxDur {
		HandleError(w, r, metric.Name, "Requested time window exceeds maximum allowed duration", http.StatusBadRequest)
		return
	}

	ok = true
	return
}

func (h *KromgoHandler) handleHistory(w http.ResponseWriter, r *http.Request, metric configuration.Metric) {
	start, end, step, ok := h.validateHistoryAccess(w, r, metric)
	if !ok {
		return
	}

	result, warnings, err := prometheus.Papi.QueryRange(r.Context(), metric.Query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error executing history query")
		HandleError(w, r, metric.Name, "Query Error", http.StatusInternalServerError)
		return
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			requestLog(r).With(zap.String("warning", warning)).Warn("encountered warnings while executing history query")
		}
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		requestLog(r).Error("history query did not return a matrix")
		HandleError(w, r, metric.Name, "Unexpected result type", http.StatusInternalServerError)
		return
	}

	title := metric.Name
	if metric.Title != "" {
		title = metric.Title
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
		Title:  title,
		Start:  start.Unix(),
		End:    end.Unix(),
		Step:   int64(step.Seconds()),
		Series: series,
	}

	jsonResponse, err := json.Marshal(resp)
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error marshaling history response")
		HandleError(w, r, metric.Name, "Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
