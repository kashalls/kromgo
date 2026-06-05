package kromgo

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeMatrix(series [][]float64) model.Matrix {
	matrix := make(model.Matrix, len(series))
	for i, values := range series {
		stream := &model.SampleStream{
			Metric: model.Metric{"series": model.LabelValue(fmt.Sprintf("s%d", i))},
			Values: make([]model.SamplePair, len(values)),
		}
		for j, v := range values {
			stream.Values[j] = model.SamplePair{
				Timestamp: model.Time(j * 60 * 1000),
				Value:     model.SampleValue(v),
			}
		}
		matrix[i] = stream
	}
	return matrix
}

func TestRenderChart_SVG(t *testing.T) {
	t.Parallel()
	svg, err := renderChart(makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 400, height: 150, legend: true, format: formatSVG})
	require.NoError(t, err)
	// Explicit width/height so <img> embeds render at the requested size.
	assert.Contains(t, string(svg), `<svg width="400" height="150"`)
}

func TestRenderChart_Title(t *testing.T) {
	t.Parallel()
	svg, err := renderChart(makeMatrix([][]float64{{1, 2, 3}}),
		chartParams{width: 400, height: 150, title: "CPU usage", format: formatSVG})
	require.NoError(t, err)
	assert.Contains(t, string(svg), "CPU usage")
}

func TestRenderChart_ValueFormatter(t *testing.T) {
	t.Parallel()
	// A graph's valueExpr becomes the y-axis ValueFormatter; a distinctive suffix on
	// every tick label proves it reaches the rendered SVG.
	svg, err := renderChart(makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 400, height: 150, format: formatSVG,
			valueFormatter: func(f float64) string { return fmt.Sprintf("%dpods", int(f)) }})
	require.NoError(t, err)
	assert.Contains(t, string(svg), "pods", "y-axis ticks are formatted by valueFormatter")

	// Without a formatter the marker is absent (the library's default numeric labels).
	plain, err := renderChart(makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 400, height: 150, format: formatSVG})
	require.NoError(t, err)
	assert.NotContains(t, string(plain), "pods")
}

func TestRenderChart_NiceYAxisTicks(t *testing.T) {
	t.Parallel()
	// Over a non-round data range (e.g. [25.3, 46.94]) the y-axis ticks must still be
	// round numbers, not an even division like 25/29.39/33.78. Otherwise a
	// full-precision valueExpr such as `string(result)` prints labels like "46.9405".
	// A %g formatter (shortest float repr, like string()/humanizeFloat) makes any
	// non-round tick visible as a long-decimal label.
	full := func(f float64) string { return fmt.Sprintf("%g", f) }
	svg, err := renderChart(makeMatrix([][]float64{{25.3, 46.94, 33.1, 41.5, 28.4, 44.0}}),
		chartParams{width: 600, height: 200, format: formatSVG, valueFormatter: full})
	require.NoError(t, err)
	assert.NotRegexp(t, `>\d+\.\d{3,}<`, string(svg),
		"y-axis ticks should be round, not full-precision floats like 46.9405")
}

// TestRenderGraph_ConfigTable renders representative graph configs end-to-end
// (config → resolveGraph → renderChart) and checks the y-axis labels: each
// valueExpr's units appear, and — whatever the formatter — ticks stay round (no
// full-precision floats), guarding the formatting regressions this package has hit.
func TestRenderGraph_ConfigTable(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)

	cpu := [][]float64{{25.3, 46.94, 33.1, 41.5, 28.4, 44.0}} // non-round CPU% range
	mem := [][]float64{{1.5e9, 2.1e9, 1.2e9, 1.8e9, 2.4e9}}   // bytes

	cases := []struct {
		name      string
		valueExpr string
		data      [][]float64
		want      []string // substrings that must appear in the rendered SVG
	}{
		{"default formatter", "", cpu, nil},
		{"percent via string", `string(result) + "%"`, cpu, []string{"%"}},
		{"percent via int", `string(int(result)) + "%"`, cpu, []string{"%"}},
		{"humanized bytes", `humanizeBytes(result)`, mem, []string{"GB"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rg, err := resolveGraph(config.Graph{ID: "g", Query: "q", ValueExpr: tc.valueExpr}, config.Defaults{}, env)
			require.NoError(t, err)
			svg, err := renderChart(makeMatrix(tc.data), rg.defaults)
			require.NoError(t, err)
			for _, w := range tc.want {
				assert.Contains(t, string(svg), w)
			}
			assert.NotRegexp(t, `>\d+\.\d{3,}`, string(svg),
				"y-axis labels should be round, not full-precision floats")
		})
	}
}

func TestRenderChart_FillArea(t *testing.T) {
	t.Parallel()
	data := makeMatrix([][]float64{{10, 25, 15, 40, 30}})

	// fill draws a translucent area (rgba) beneath the line.
	filled, err := renderChart(data, chartParams{width: 400, height: 150, format: formatSVG, fill: true})
	require.NoError(t, err)
	assert.Contains(t, string(filled), "fill:rgba", "fill should draw a translucent area")

	// Without fill, only the line stroke is drawn — no area fill.
	plain, err := renderChart(data, chartParams{width: 400, height: 150, format: formatSVG})
	require.NoError(t, err)
	assert.NotContains(t, string(plain), "fill:rgba", "no area fill when fill is off")
}

func TestRenderChart_PNG(t *testing.T) {
	t.Parallel()
	png, err := renderChart(makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 400, height: 150, format: formatPNG})
	require.NoError(t, err)
	// PNG magic number.
	require.GreaterOrEqual(t, len(png), 8)
	assert.Equal(t, []byte{0x89, 'P', 'N', 'G'}, png[:4])
}

func TestRenderChart_NaNAndInfBecomeGaps(t *testing.T) {
	t.Parallel()
	// Non-finite samples must not produce a broken image or literal NaN/Inf text.
	matrix := makeMatrix([][]float64{{10, 20, 30, 40}})
	matrix[0].Values[1].Value = model.SampleValue(math.NaN())
	matrix[0].Values[2].Value = model.SampleValue(math.Inf(1))

	svg, err := renderChart(matrix, chartParams{width: 400, height: 150, format: formatSVG})
	require.NoError(t, err)
	assert.NotContains(t, string(svg), "NaN")
	assert.NotContains(t, string(svg), "Inf")
}

func TestRenderChart_Theme(t *testing.T) {
	t.Parallel()
	// A custom theme's background color should appear in the rendered SVG.
	matrix := makeMatrix([][]float64{{1, 2, 3}})
	svg, err := renderChart(matrix, chartParams{width: 400, height: 150, theme: "dracula", format: formatSVG})
	require.NoError(t, err)
	// go-analyze/charts emits colors as rgb(); dracula's #282a36 background = rgb(40,42,54).
	assert.Contains(t, string(svg), "rgb(40,42,54)", "dracula background should be present")
}

func TestSeriesLabel(t *testing.T) {
	t.Parallel()
	stream := &model.SampleStream{Metric: model.Metric{
		"__name__": "x", "instance": "node-1", "job": "kube",
	}}
	// Sorted by key (instance, job); __name__ excluded.
	assert.Equal(t, "node-1, kube", seriesLabel(stream))

	bare := &model.SampleStream{Metric: model.Metric{"__name__": "x"}}
	assert.Empty(t, seriesLabel(bare))
}

func TestChartParams_WithOverrides(t *testing.T) {
	t.Parallel()
	base := chartParams{width: 300, height: 80, legend: true, theme: "dark", format: formatSVG}

	req := httptest.NewRequest(http.MethodGet,
		"/?width=500&height=250&legend=false&fill=true&theme=dracula&format=png", nil)
	got := base.withOverrides(req)

	assert.Equal(t, 500, got.width)
	assert.Equal(t, 250, got.height)
	assert.False(t, got.legend)
	assert.True(t, got.fill)
	assert.Equal(t, "dracula", got.theme)
	assert.Equal(t, formatPNG, got.format)
	assert.Equal(t, "image/png", got.contentType())

	// Width is clamped to the maximum.
	clamped := base.withOverrides(httptest.NewRequest(http.MethodGet, "/?width=99999", nil))
	assert.Equal(t, maxChartDimension, clamped.width)
}

func TestResolveGraphFont(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"", "dejavu-sans", "dejavu-sans-bold"} {
		t.Run("ok/"+name, func(t *testing.T) {
			t.Parallel()
			f, err := resolveGraphFont(name)
			require.NoError(t, err)
			assert.NotNil(t, f, "empty name defaults to DejaVu Sans")
		})
	}
	// An unknown font name errors (no disk fallback).
	t.Run("unknown errors", func(t *testing.T) {
		t.Parallel()
		_, err := resolveGraphFont("not-a-font")
		assert.Error(t, err)
	})
}

func TestRenderChart_CustomFont(t *testing.T) {
	t.Parallel()
	font, err := resolveGraphFont("dejavu-sans-bold")
	require.NoError(t, err)
	svg, err := renderChart(makeMatrix([][]float64{{1, 2, 3}}),
		chartParams{width: 400, height: 150, font: font, format: formatSVG})
	require.NoError(t, err)
	assert.Contains(t, string(svg), "<svg")
}

func TestValidTheme(t *testing.T) {
	t.Parallel()
	cases := []struct {
		theme string
		want  bool
	}{
		{"dark", true},             // built-in
		{"grafana", true},          // built-in
		{"dracula", true},          // custom
		{"catppuccin-mocha", true}, // custom
		{"nope", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.theme, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, validTheme(tc.theme))
		})
	}
}
