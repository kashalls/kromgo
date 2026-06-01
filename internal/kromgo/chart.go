package kromgo

import (
	"fmt"
	"html"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/prometheus/common/model"
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

const (
	maxChartDimension = 2048
	maxStrokeWidth    = 20.0
)

// withOverrides returns the graph's resolved default params with any request query
// parameters applied on top (width/height/stroke/color/legend).
func (p chartParams) withOverrides(r *http.Request) chartParams {
	q := r.URL.Query()
	if s := q.Get("width"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			p.width = min(v, maxChartDimension)
		}
	}
	if s := q.Get("height"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			p.height = min(v, maxChartDimension)
		}
	}
	if s := q.Get("stroke"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			p.strokeWidth = min(v, maxStrokeWidth)
		}
	}
	if s := q.Get("color"); s != "" {
		p.color = s
	}
	switch q.Get("legend") {
	case "false":
		p.legend = false
	case "true":
		p.legend = true
	}
	return p
}

func seriesColor(i int, override string) string {
	if override != "" {
		return colorNameToHex(override)
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
	slices.Sort(keys)
	vals := make([]string, 0, len(keys))
	for _, k := range keys {
		vals = append(vals, string(stream.Metric[model.LabelName(k)]))
	}
	return strings.Join(vals, ", ")
}

func renderSparkline(matrix model.Matrix, p chartParams) string {
	const (
		legendHeight       = 20
		legendFontSize     = 11
		legendIndicatorW   = 16
		legendIndicatorGap = 5
		legendItemMargin   = 12
		legendCharWidth    = 6.5
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
					color: seriesColor(i, p.color),
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

	type point struct{ x, y float64 }
	for i, stream := range matrix {
		// Prometheus can return NaN/Inf samples (counter resets, staleness gaps);
		// they would poison min/max and emit NaN SVG coordinates, so skip them. A
		// single pass collects each finite sample's x (from its original index, so
		// skipped samples leave a gap) and raw value while tracking min/max; y is
		// finalized below once the range is known.
		n := len(stream.Values)
		pts := make([]point, 0, n)
		minVal := math.Inf(1)
		maxVal := math.Inf(-1)
		for j, pt := range stream.Values {
			v := float64(pt.Value)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			minVal = min(minVal, v)
			maxVal = max(maxVal, v)
			pts = append(pts, point{x: pad + float64(j)/float64(max(n-1, 1))*w, y: v})
		}
		if len(pts) == 0 {
			continue // no finite samples to plot
		}
		valRange := maxVal - minVal
		if valRange == 0 {
			valRange = 1
		}
		for k := range pts {
			pts[k].y = pad + (1-(pts[k].y-minVal)/valRange)*h
		}

		color := seriesColor(i, p.color)

		// Filled area under the line.
		var area strings.Builder
		fmt.Fprintf(&area, "M %.2f,%.2f", pts[0].x, pts[0].y)
		for _, pt := range pts[1:] {
			fmt.Fprintf(&area, " L %.2f,%.2f", pt.x, pt.y)
		}
		fmt.Fprintf(&area, " L %.2f,%.2f L %.2f,%.2f Z", pts[len(pts)-1].x, bottom, pts[0].x, bottom)
		fmt.Fprintf(&sb, `<path d="%s" fill="%s" fill-opacity="0.15" stroke="none"/>`, area.String(), color)

		// Line.
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

	// Legend row.
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
