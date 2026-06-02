package kromgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorScale(t *testing.T) {
	t.Parallel()
	steps := []float64{35, 75}
	colors := []string{"green", "orange", "red"}
	cases := []struct {
		in   float64
		want string
	}{
		{10, "green"}, {34.9, "green"}, {35, "orange"}, {74.9, "orange"},
		{75, "red"}, {1000, "red"},
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want, colorScale(tc.in, steps, colors), "colorScale(%v)", tc.in)
	}

	// Misconfigured: colors must be exactly one longer than steps → "grey", not a panic.
	assert.Equal(t, "grey", colorScale(50, steps, []string{"green", "red"}))
	assert.Equal(t, "grey", colorScale(50, steps, nil))
}
