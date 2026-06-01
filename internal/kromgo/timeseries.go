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

// The helpers below are the windowed range-query foundation shared by the graph
// output formats: parameter parsing, window validation, and the range query itself.

var (
	errStartAfterEnd       = &graphParamError{"start must be before end"}
	errNonPositiveDuration = &graphParamError{"last must be a positive duration"}
)

type graphParamError struct{ msg string }

func (e *graphParamError) Error() string { return e.msg }

func parseTimeParam(s string) (time.Time, error) {
	// Try Unix timestamp (integer) first, then fall back to RFC3339.
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	return time.Parse(time.RFC3339, s)
}

func parseGraphParams(r *http.Request) (start, end time.Time, step time.Duration, err error) {
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

	step = autoStep(end.Sub(start))
	if s := q.Get("step"); s != "" {
		if step, err = config.ParseDuration(s); err != nil {
			return start, end, step, err
		}
		step = max(step, minRangeStep)
	}

	return start, end, step, nil
}

// validateGraphAccess parses the time parameters and enforces the graph's max-duration
// cap. Returns ok=false if an error response was already written.
func (h *Handler) validateGraphAccess(w http.ResponseWriter, r *http.Request, graph *resolvedGraph) (start, end time.Time, step time.Duration, ok bool) {
	start, end, step, err := parseGraphParams(r)
	if err != nil {
		writeError(w, graph.ID, "Invalid parameter: "+err.Error(), http.StatusBadRequest)
		return start, end, step, false
	}

	if graph.maxDuration > 0 && end.Sub(start) > graph.maxDuration {
		writeError(w, graph.ID, "Requested time window exceeds maximum allowed duration", http.StatusBadRequest)
		return start, end, step, false
	}

	return start, end, step, true
}

// queryMatrix runs a range query and asserts the result is a matrix, writing an error
// response and returning ok=false otherwise.
func (h *Handler) queryMatrix(w http.ResponseWriter, r *http.Request, graph *resolvedGraph, start, end time.Time, step time.Duration, log *slog.Logger) (model.Matrix, bool) {
	value, err := h.prom.QueryRange(r.Context(), graph.Query, v1.Range{Start: start, End: end, Step: step})
	if err != nil {
		log.Error("error executing range query", "error", err)
		writeError(w, graph.ID, "Query Error", http.StatusInternalServerError)
		return nil, false
	}
	matrix, ok := value.(model.Matrix)
	if !ok {
		log.Error("range query did not return a matrix", "type", value.Type().String())
		writeError(w, graph.ID, "Unexpected result type", http.StatusInternalServerError)
		return nil, false
	}
	return matrix, true
}
