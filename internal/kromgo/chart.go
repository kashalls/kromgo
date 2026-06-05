package kromgo

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	charts "github.com/go-analyze/charts"
	"github.com/golang/freetype/truetype"
	"github.com/prometheus/common/model"
)

const (
	maxChartDimension = 2048
	formatPNG         = "png"
	// fillOpacity is the alpha (0-255) for the area fill — translucent so overlapping
	// per-series fills stay legible (the library's default is a heavier 200).
	fillOpacity = 90
)

// chartParams holds the resolved sparkline rendering parameters. The graph's
// config provides the defaults; request query parameters override them.
type chartParams struct {
	width  int
	height int
	legend bool
	theme  string
	title  string         // chart title, rendered top-left
	font   *truetype.Font // nil uses the chart library's default font
	format string         // "svg" (default) or "png"
	fill   bool           // draw a translucent area beneath the line(s)
	// valueFormatter renders y-axis tick values to strings (from the graph's
	// valueExpr). nil uses the chart library's default numeric formatting.
	valueFormatter func(float64) string
}

// withOverrides returns the graph's default params with request query parameters
// applied on top (width/height/legend/theme/format).
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
	switch q.Get("legend") {
	case "false":
		p.legend = false
	case "true":
		p.legend = true
	}
	switch q.Get("fill") {
	case "false":
		p.fill = false
	case "true":
		p.fill = true
	}
	if s := q.Get("theme"); s != "" {
		p.theme = s // unknown names fall back to the default in chartTheme
	}
	if q.Get("format") == formatPNG {
		p.format = formatPNG
	}
	return p
}

// contentType returns the MIME type for the params' output format.
func (p chartParams) contentType() string {
	if p.format == formatPNG {
		return mimePNG
	}
	return mimeSVG
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

// renderChart draws the matrix as a themed line chart and returns the encoded image
// (SVG or PNG). Non-finite samples (NaN/Inf) become gaps.
func renderChart(matrix model.Matrix, p chartParams) ([]byte, error) {
	values := make([][]float64, len(matrix))
	labels := make([]string, len(matrix))
	haveLabels := false
	var xAxis []string

	for i, stream := range matrix {
		row := make([]float64, len(stream.Values))
		for j, pt := range stream.Values {
			v := float64(pt.Value)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = charts.GetNullValue() // rendered as a gap
			}
			row[j] = v
		}
		values[i] = row
		if label := seriesLabel(stream); label != "" {
			// Escape: the charting library writes legend labels into SVG <text>
			// without escaping, so a metric label value could inject markup/script.
			labels[i] = html.EscapeString(label)
			haveLabels = true
		}
		if xAxis == nil {
			xAxis = timeAxisLabels(stream.Values)
		}
	}

	opt := charts.NewLineChartOptionWithData(values)
	opt.Theme = chartTheme(p.theme)
	opt.XAxis.Labels = xAxis
	// One label per sample collides; cap the count by width so the library samples
	// an evenly-spaced, non-overlapping subset.
	if n := len(xAxis); n > 0 {
		opt.XAxis.LabelCount = min(max(p.width/110, 2), n)
	}
	if p.title != "" {
		opt.Title = charts.TitleOption{Text: p.title, Offset: charts.OffsetLeft}
	}
	if p.legend && haveLabels {
		opt.Legend.SeriesNames = labels
	} else {
		hide := false
		opt.Legend.Show = &hide
	}
	// Fill the area beneath the line(s). The library's default fill alpha (200/255) is
	// heavy when kromgo's per-series lines overlap, so use a lighter, translucent value.
	if p.fill {
		fill := true
		opt.FillArea = &fill
		opt.FillOpacity = fillOpacity
	}
	// Ask the chart library for round y-axis tick values (e.g. 25/30/35/40/45 rather
	// than dividing the range evenly into 25/29.39/33.78/…). Without this the ticks
	// land on arbitrary floats, which a valueExpr like `string(result)` then prints at
	// full precision ("46.9405%"). A graph's valueExpr (if any) then formats those
	// values — integers or humanized units in place of the default 2-decimal numbers.
	niceIntervals := true
	yAxis := charts.YAxisOption{PreferNiceIntervals: &niceIntervals}
	if p.valueFormatter != nil {
		yAxis.ValueFormatter = p.valueFormatter
	}
	opt.YAxis = []charts.YAxisOption{yAxis}

	// Font is set on the painter (the non-deprecated default-font hook). resolveGraphFont
	// always returns a face (DejaVu Sans by default), so p.font is never nil here.
	painter := charts.NewPainter(charts.PainterOptions{
		OutputFormat: p.format, // "svg" or "png"
		Width:        p.width,
		Height:       p.height,
		Font:         p.font,
	})
	if err := painter.LineChart(opt); err != nil {
		return nil, err
	}
	out, err := painter.Bytes()
	if err != nil {
		return nil, err
	}
	if p.format != formatPNG {
		// The chart library emits SVG with only a viewBox; add explicit width/height
		// so <img> embeds (and inline use) render at the requested pixel size rather
		// than the browser's 300x150 default.
		dims := fmt.Sprintf(`<svg width="%d" height="%d" `, p.width, p.height)
		out = bytes.Replace(out, []byte("<svg "), []byte(dims), 1)
	}
	return out, nil
}

// timeAxisLabels formats one x-axis label per sample; the chart library samples
// them to avoid crowding. The format adapts to the window span.
func timeAxisLabels(values []model.SamplePair) []string {
	if len(values) == 0 {
		return nil
	}
	span := values[len(values)-1].Timestamp.Time().Sub(values[0].Timestamp.Time())
	layout := "15:04"
	if span >= 24*time.Hour {
		layout = "01/02"
	}
	labels := make([]string, len(values))
	for i, pt := range values {
		labels[i] = pt.Timestamp.Time().Format(layout)
	}
	return labels
}
