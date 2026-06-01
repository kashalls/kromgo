package kromgo

import (
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const xssPayload = `</text><script>alert(1)</script>`

// Badge text comes from a CEL value or a metric label; it must never reach the SVG
// unescaped, or opening the badge as a top-level document would execute script.
func TestSecurity_BadgeEscapesSVG(t *testing.T) {
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	body := string(r.render(config.StyleFlat, "", "label", xssPayload, "green"))
	assert.NotContains(t, body, "<script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

// Graph legend labels come from metric label values and must be escaped too.
func TestSecurity_GraphLegendEscapesSVG(t *testing.T) {
	matrix := model.Matrix{&model.SampleStream{
		Metric: model.Metric{"instance": model.LabelValue(xssPayload)},
		Values: []model.SamplePair{{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}},
	}}

	svg, err := renderChart(matrix, chartParams{width: 400, height: 200, legend: true, format: formatSVG})
	require.NoError(t, err)

	assert.NotContains(t, string(svg), "<script>")
}
