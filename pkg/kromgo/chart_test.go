package kromgo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/common/model"
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

func makeMatrixLabeled(series map[string][]float64) model.Matrix {
	matrix := make(model.Matrix, 0, len(series))
	for name, values := range series {
		stream := &model.SampleStream{
			Metric: model.Metric{"instance": model.LabelValue(name)},
			Values: make([]model.SamplePair, len(values)),
		}
		for j, v := range values {
			stream.Values[j] = model.SamplePair{
				Timestamp: model.Time(j * 60 * 1000),
				Value:     model.SampleValue(v),
			}
		}
		matrix = append(matrix, stream)
	}
	return matrix
}

func TestRenderSparkline_Structure(t *testing.T) {
	svg := renderSparkline(
		makeMatrix([][]float64{{10, 25, 15, 40, 30}}),
		chartParams{width: 300, height: 80, strokeWidth: 2, legend: true},
		nil,
	)
	if !strings.HasPrefix(svg, "<svg ") {
		t.Error("expected SVG to start with <svg")
	}
	if !strings.HasSuffix(svg, "</svg>") {
		t.Error("expected SVG to end with </svg>")
	}
}

func TestRenderSparkline_PolylineCount(t *testing.T) {
	cases := []struct {
		name     string
		matrix   model.Matrix
		wantLines int
	}{
		{"single", makeMatrix([][]float64{{1, 2, 3}}), 1},
		{"two_series", makeMatrix([][]float64{{1, 2}, {3, 4}}), 2},
		{"empty_series_skipped", func() model.Matrix {
			m := makeMatrix([][]float64{{1, 2}, {}})
			return m
		}(), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svg := renderSparkline(tc.matrix, chartParams{width: 300, height: 80, strokeWidth: 2}, nil)
			got := strings.Count(svg, "<polyline ")
			if got != tc.wantLines {
				t.Errorf("expected %d <polyline> elements, got %d", tc.wantLines, got)
			}
		})
	}
}

func TestRenderSparkline_LegendText(t *testing.T) {
	matrix := makeMatrixLabeled(map[string][]float64{
		"server1": {10, 20, 30},
		"server2": {40, 50, 60},
	})

	withLegend := renderSparkline(matrix, chartParams{width: 300, height: 80, strokeWidth: 2, legend: true}, nil)
	if !strings.Contains(withLegend, "server1") || !strings.Contains(withLegend, "server2") {
		t.Error("expected legend labels in SVG when legend=true")
	}

	withoutLegend := renderSparkline(matrix, chartParams{width: 300, height: 80, strokeWidth: 2, legend: false}, nil)
	if strings.Contains(withoutLegend, "<text ") {
		t.Error("expected no <text> elements when legend=false")
	}
}

func TestRenderSparkline_FlatLineNoNaN(t *testing.T) {
	svg := renderSparkline(
		makeMatrix([][]float64{{42, 42, 42, 42, 42}}),
		chartParams{width: 300, height: 80, strokeWidth: 2},
		nil,
	)
	if strings.Contains(svg, "NaN") {
		t.Error("SVG contains NaN coordinates")
	}
	if strings.Contains(svg, "Inf") {
		t.Error("SVG contains Inf coordinates")
	}
}
