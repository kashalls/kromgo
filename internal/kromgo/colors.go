package kromgo

import "math"

// fallbackColor is the shields.io color name colorScale returns when it can't map a
// value to a band — a NaN value, or a misconfigured steps/colors length.
const fallbackColor = "grey"

// colorScale is registered with the CEL environment (see expr.go) for a metric's
// colorExpr, e.g. colorExpr: colorScale(result, [35.0, 75.0], ["green","orange","red"]).
// It maps a value to a shields.io color name (resolved to hex later by colorNameToHex),
// so threshold coloring isn't a hand-written chain of ternaries.
//
// It returns colors[i] for the first step where value < steps[i], otherwise the last
// color; colors must hold exactly one more entry than steps. A NaN value (or a
// misconfigured call) returns "grey" rather than a color — kromgo degrades color, it
// never 500s on it.
func colorScale(value float64, steps []float64, colors []string) string {
	if len(colors) != len(steps)+1 {
		return fallbackColor
	}
	// NaN < step is always false, so a NaN would otherwise fall through every
	// threshold to the last (worst) color — silently painting "missing" as "critical".
	if math.IsNaN(value) {
		return fallbackColor
	}
	for i, step := range steps {
		if value < step {
			return colors[i]
		}
	}
	return colors[len(colors)-1]
}
