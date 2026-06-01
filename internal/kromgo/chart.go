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
)

// chartParams holds the resolved sparkline rendering parameters. The graph's
// config provides the defaults; request query parameters override them.
type chartParams struct {
	width  int
	height int
	legend bool
	theme  string
	font   *truetype.Font // nil uses the chart library's default font
	format string         // "svg" (default) or "png"
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
	if p.legend && haveLabels {
		opt.Legend.SeriesNames = labels
	} else {
		hide := false
		opt.Legend.Show = &hide
	}

	// Font is set on the painter (the non-deprecated default-font hook); nil leaves
	// the chart library's default.
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
