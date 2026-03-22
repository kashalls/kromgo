package kromgo

import (
	"fmt"
	"html"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

var chartColors = []string{
	"#4c8bcd", "#e05e4c", "#4ccd8b", "#e0c04c", "#8b4ccd",
}

type chartParams struct {
	width       int
	height      int
	strokeWidth float64
	color       string
	legend      bool
}

const maxChartDimension = 2048
const maxStrokeWidth = 20.0

func parseChartParams(r *http.Request) chartParams {
	p := chartParams{width: 300, height: 80, strokeWidth: 2, legend: true}
	if s := r.URL.Query().Get("width"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			p.width = min(v, maxChartDimension)
		}
	}
	if s := r.URL.Query().Get("height"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			p.height = min(v, maxChartDimension)
		}
	}
	if s := r.URL.Query().Get("stroke"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			p.strokeWidth = min(v, maxStrokeWidth)
		}
	}
	p.color = r.URL.Query().Get("color")
	if r.URL.Query().Get("legend") == "false" {
		p.legend = false
	}
	return p
}

func seriesColor(i int, override string, colors []configuration.MetricColor) string {
	if override != "" {
		return colorNameToHex(override)
	}
	if i == 0 && len(colors) > 0 && colors[0].Color != "" {
		return colorNameToHex(colors[0].Color)
	}
	return chartColors[i%len(chartColors)]
}

// seriesLabel returns a display label for a series by joining its non-__name__ label values.
func seriesLabel(stream *model.SampleStream) string {
	keys := make([]string, 0, len(stream.Metric))
	for k := range stream.Metric {
		if k != "__name__" {
			keys = append(keys, string(k))
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	vals := make([]string, 0, len(keys))
	for _, k := range keys {
		vals = append(vals, string(stream.Metric[model.LabelName(k)]))
	}
	return strings.Join(vals, ", ")
}

func renderSparkline(matrix model.Matrix, p chartParams, metricColors []configuration.MetricColor) string {
	const (
		legendHeight        = 20
		legendFontSize      = 11
		legendIndicatorW    = 16
		legendIndicatorGap  = 5
		legendItemMargin    = 12
		legendCharWidth     = 6.5
	)

	pad := 4.0
	w := float64(p.width) - 2*pad
	h := float64(p.height) - 2*pad
	bottom := pad + h

	// Build legend items before rendering so we know total SVG height.
	type legendItem struct {
		color string
		label string
	}
	var items []legendItem
	if p.legend {
		for i, stream := range matrix {
			if label := seriesLabel(stream); label != "" {
				items = append(items, legendItem{
					color: seriesColor(i, p.color, metricColors),
					label: label,
				})
			}
		}
	}

	totalHeight := p.height
	if len(items) > 0 {
		totalHeight += legendHeight
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		p.width, totalHeight, p.width, totalHeight)

	for i, stream := range matrix {
		if len(stream.Values) == 0 {
			continue
		}

		minVal := math.Inf(1)
		maxVal := math.Inf(-1)
		for _, pt := range stream.Values {
			v := float64(pt.Value)
			minVal = min(minVal, v)
			maxVal = max(maxVal, v)
		}
		valRange := maxVal - minVal
		if valRange == 0 {
			valRange = 1
		}

		n := len(stream.Values)
		color := seriesColor(i, p.color, metricColors)

		type point struct{ x, y float64 }
		pts := make([]point, n)
		for j, pt := range stream.Values {
			pts[j] = point{
				x: pad + float64(j)/float64(max(n-1, 1))*w,
				y: pad + (1-(float64(pt.Value)-minVal)/valRange)*h,
			}
		}

		// Filled area under the line
		var area strings.Builder
		fmt.Fprintf(&area, "M %.2f,%.2f", pts[0].x, pts[0].y)
		for _, pt := range pts[1:] {
			fmt.Fprintf(&area, " L %.2f,%.2f", pt.x, pt.y)
		}
		fmt.Fprintf(&area, " L %.2f,%.2f L %.2f,%.2f Z", pts[n-1].x, bottom, pts[0].x, bottom)
		fmt.Fprintf(&sb, `<path d="%s" fill="%s" fill-opacity="0.15" stroke="none"/>`, area.String(), color)

		// Line
		var line strings.Builder
		for j, pt := range pts {
			if j > 0 {
				line.WriteByte(' ')
			}
			fmt.Fprintf(&line, "%.2f,%.2f", pt.x, pt.y)
		}
		fmt.Fprintf(&sb, `<polyline points="%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-linecap="round" stroke-linejoin="round"/>`,
			line.String(), color, p.strokeWidth)
	}

	// Legend row
	if len(items) > 0 {
		lineY := float64(p.height) + float64(legendHeight)/2
		textY := lineY + float64(legendFontSize)/2 - 1
		x := pad
		for _, item := range items {
			fmt.Fprintf(&sb, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="2" stroke-linecap="round"/>`,
				x, lineY, x+legendIndicatorW, lineY, item.color)
			x += legendIndicatorW + legendIndicatorGap
			fmt.Fprintf(&sb, `<text x="%.2f" y="%.2f" font-family="sans-serif" font-size="%d" fill="#666">%s</text>`,
				x, textY, legendFontSize, html.EscapeString(item.label))
			x += float64(len(item.label))*legendCharWidth + legendItemMargin
		}
	}

	sb.WriteString("</svg>")
	return sb.String()
}

func (h *KromgoHandler) handleChart(w http.ResponseWriter, r *http.Request, metric configuration.Metric) {
	start, end, step, ok := h.validateHistoryAccess(w, r, metric)
	if !ok {
		return
	}

	cp := parseChartParams(r)

	result, warnings, err := prometheus.Papi.QueryRange(r.Context(), metric.Query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error executing chart query")
		HandleError(w, r, metric.Name, "Query Error", http.StatusInternalServerError)
		return
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			requestLog(r).With(zap.String("warning", warning)).Warn("encountered warnings while executing chart query")
		}
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		requestLog(r).Error("chart query did not return a matrix")
		HandleError(w, r, metric.Name, "Unexpected result type", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write([]byte(renderSparkline(matrix, cp, metric.Colors)))
}
