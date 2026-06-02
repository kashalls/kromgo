package kromgo

// colorScale is registered with the CEL environment (see expr.go) for a metric's
// colorExpr, e.g. colorExpr: colorScale(result, [35.0, 75.0], ["green","orange","red"]).
// It maps a value to a shields.io color name (resolved to hex later by colorNameToHex),
// so threshold coloring isn't a hand-written chain of ternaries.
//
// It returns colors[i] for the first step where value < steps[i], otherwise the last
// color; colors must hold exactly one more entry than steps. A misconfigured call
// returns "grey" rather than erroring (kromgo degrades color, it never 500s on it).
func colorScale(value float64, steps []float64, colors []string) string {
	if len(colors) != len(steps)+1 {
		return "grey"
	}
	for i, step := range steps {
		if value < step {
			return colors[i]
		}
	}
	return colors[len(colors)-1]
}
