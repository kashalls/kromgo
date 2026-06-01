package kromgo

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

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
	svg, err := renderChart(makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 400, height: 150, legend: true, format: formatSVG})
	require.NoError(t, err)
	assert.Contains(t, string(svg), "<svg")
}

func TestRenderChart_PNG(t *testing.T) {
	png, err := renderChart(makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 400, height: 150, format: formatPNG})
	require.NoError(t, err)
	// PNG magic number.
	require.GreaterOrEqual(t, len(png), 8)
	assert.Equal(t, []byte{0x89, 'P', 'N', 'G'}, png[:4])
}

func TestRenderChart_NaNAndInfBecomeGaps(t *testing.T) {
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
	// A custom theme's background color should appear in the rendered SVG.
	matrix := makeMatrix([][]float64{{1, 2, 3}})
	svg, err := renderChart(matrix, chartParams{width: 400, height: 150, theme: "dracula", format: formatSVG})
	require.NoError(t, err)
	// go-analyze/charts emits colors as rgb(); dracula's #282a36 background = rgb(40,42,54).
	assert.Contains(t, string(svg), "rgb(40,42,54)", "dracula background should be present")
}

func TestSeriesLabel(t *testing.T) {
	stream := &model.SampleStream{Metric: model.Metric{
		"__name__": "x", "instance": "node-1", "job": "kube",
	}}
	// Sorted by key (instance, job); __name__ excluded.
	assert.Equal(t, "node-1, kube", seriesLabel(stream))

	bare := &model.SampleStream{Metric: model.Metric{"__name__": "x"}}
	assert.Empty(t, seriesLabel(bare))
}

func TestChartParams_WithOverrides(t *testing.T) {
	base := chartParams{width: 300, height: 80, legend: true, theme: "dark", format: formatSVG}

	req := httptest.NewRequest(http.MethodGet,
		"/?width=500&height=250&legend=false&theme=dracula&format=png", nil)
	got := base.withOverrides(req)

	assert.Equal(t, 500, got.width)
	assert.Equal(t, 250, got.height)
	assert.False(t, got.legend)
	assert.Equal(t, "dracula", got.theme)
	assert.Equal(t, formatPNG, got.format)
	assert.Equal(t, "image/png", got.contentType())

	// Width is clamped to the maximum.
	clamped := base.withOverrides(httptest.NewRequest(http.MethodGet, "/?width=99999", nil))
	assert.Equal(t, maxChartDimension, clamped.width)
}

func TestResolveGraphFont(t *testing.T) {
	for _, name := range []string{"", "roboto", "notosans", "go-regular", "go-bold", "go-medium", "go-mono"} {
		t.Run("ok/"+name, func(t *testing.T) {
			f, err := resolveGraphFont(name)
			require.NoError(t, err)
			if name == "" {
				assert.Nil(t, f, "empty name uses the library default")
			} else {
				assert.NotNil(t, f)
			}
		})
	}
	// An unknown name is treated as a (missing) file path and errors.
	_, err := resolveGraphFont("not-a-font")
	assert.Error(t, err)
}

func TestRenderChart_CustomFont(t *testing.T) {
	font, err := resolveGraphFont("go-bold")
	require.NoError(t, err)
	svg, err := renderChart(makeMatrix([][]float64{{1, 2, 3}}),
		chartParams{width: 400, height: 150, font: font, format: formatSVG})
	require.NoError(t, err)
	assert.Contains(t, string(svg), "<svg")
}

func TestValidTheme(t *testing.T) {
	assert.True(t, validTheme("dark"))             // built-in
	assert.True(t, validTheme("grafana"))          // built-in
	assert.True(t, validTheme("dracula"))          // custom
	assert.True(t, validTheme("catppuccin-mocha")) // custom
	assert.False(t, validTheme("nope"))
	assert.False(t, validTheme(""))
}
