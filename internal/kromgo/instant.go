package kromgo

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

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
