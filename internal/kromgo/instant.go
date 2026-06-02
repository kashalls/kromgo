package kromgo

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/home-operations/kromgo/internal/logging"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// BadgeJSON is kromgo's native JSON for a badge value (format=json): the rendered
// string plus the underlying number and labels, without the Prometheus envelope.
type BadgeJSON struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Value  string            `json:"value"`
	Color  string            `json:"color,omitempty"`
	Result *float64          `json:"result,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

// serveBadge renders an instant value as an SVG badge (default), shields.io endpoint
// JSON (?format=shields), or kromgo JSON (?format=json).
func (h *Handler) serveBadge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	format := r.URL.Query().Get("format")
	switch format {
	case formatJSON, formatShields: // recognized
	default:
		format = formatSVG // empty or unknown renders the badge image
	}

	metricLabel := "unknown"
	defer func() { requestsTotal.WithLabelValues("badge", metricLabel, format).Inc() }()
	log := logging.FromContext(r.Context()).With("kind", "badge", "id", id, "format", format)

	badge, ok := h.badges[id]
	if !ok {
		log.Error("badge not found")
		writeError(w, id, "Not Found", http.StatusNotFound)
		return
	}
	metricLabel = id
	h.cache.apply(w)

	value, err := h.queryValue(r.Context(), badge)
	if err != nil {
		log.Error("error executing query", "error", err)
		writeError(w, id, "Query Error", http.StatusInternalServerError)
		return
	}
	vector, ok := value.(model.Vector)
	if !ok {
		log.Error("query did not return an instant vector", "type", value.Type().String())
		writeError(w, id, "Unexpected result type", http.StatusInternalServerError)
		return
	}

	title := displayTitle(badge.Title, badge.ID)
	message, color := "no data", ""
	var result *float64
	var labels map[string]string
	if len(vector) > 0 {
		msg, col, ok := h.evalDisplay(badge, vector[0], log)
		if !ok {
			writeError(w, id, "Expression Error", http.StatusInternalServerError)
			return
		}
		v := float64(vector[0].Value)
		message, color, result, labels = msg, col, &v, labelMap(vector[0].Metric)
	}

	switch format {
	case formatShields:
		writeJSONOr(w, log, id, EndpointResponse{
			SchemaVersion: 1, Label: title, Message: message, Color: color, CacheSeconds: h.cache.seconds,
		})
	case formatJSON:
		writeJSONOr(w, log, id, BadgeJSON{
			ID: badge.ID, Title: title, Value: message, Color: color, Result: result, Labels: labels,
		})
	default: // svg
		// Label text: explicit Title, else the id — unless an icon stands in for it.
		labelText := badge.Title
		if labelText == "" && badge.iconPath == "" {
			labelText = badge.ID
		}
		style := cmp.Or(r.URL.Query().Get("style"), badge.style)
		writeSVG(w, h.gen.render(style, badge.iconPath, labelText, message, color))
	}
}

// evalDisplay evaluates the badge's value and color CEL expressions against a
// sample. ok is false only if the value expression errors (caller returns 500);
// a failing color expression is logged and treated as no color.
func (h *Handler) evalDisplay(badge *resolvedBadge, sample *model.Sample, log *slog.Logger) (message, color string, ok bool) {
	result := float64(sample.Value)
	labels := labelMap(sample.Metric)

	message, err := evalStringExpr(badge.valueProg, result, labels)
	if err != nil {
		log.Error("value expression failed", "error", err)
		return "", "", false
	}
	if badge.colorProg != nil {
		if color, err = evalStringExpr(badge.colorProg, result, labels); err != nil {
			log.Error("color expression failed", "error", err) // degrade to no color
			color = ""
		}
	}
	return message, color, true
}

// queryValue computes a badge's instant value: an instant query for the default
// type, or a range query reduced to one value per series for type: range.
func (h *Handler) queryValue(ctx context.Context, badge *resolvedBadge) (model.Value, error) {
	rq := badge.rangeQuery
	if rq == nil {
		return h.prom.Query(ctx, badge.Query, time.Now())
	}

	end := time.Now().Add(-rq.offset)
	value, err := h.prom.QueryRange(ctx, badge.Query, v1.Range{Start: end.Add(-rq.last), End: end, Step: rq.step})
	if err != nil {
		return nil, err
	}
	matrix, ok := value.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("range query returned %s, want matrix", value.Type())
	}
	return reduceMatrix(matrix, rq.reduce), nil
}
