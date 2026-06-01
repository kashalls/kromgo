package kromgo

import (
	"math"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samples(values ...float64) []model.SamplePair {
	out := make([]model.SamplePair, len(values))
	for i, v := range values {
		out[i] = model.SamplePair{Timestamp: model.Time(i * 1000), Value: model.SampleValue(v)}
	}
	return out
}

func TestReduceSamples(t *testing.T) {
	vals := samples(10, 30, 20, 40)
	cases := map[string]float64{
		config.ReduceFirst: 10,
		config.ReduceLast:  40,
		config.ReduceMin:   10,
		config.ReduceMax:   40,
		config.ReduceSum:   100,
		config.ReduceAvg:   25,
	}
	for op, want := range cases {
		t.Run(op, func(t *testing.T) {
			got, _, ok := reduceSamples(vals, op)
			require.True(t, ok)
			assert.Equal(t, want, float64(got))
		})
	}
}

func TestReduceSamples_SkipsNonFinite(t *testing.T) {
	vals := samples(10, math.NaN(), 30, math.Inf(1))
	got, _, ok := reduceSamples(vals, config.ReduceAvg)
	require.True(t, ok)
	assert.Equal(t, 20.0, float64(got)) // (10+30)/2, NaN/Inf skipped
}

func TestReduceSamples_AllNonFinite(t *testing.T) {
	_, _, ok := reduceSamples(samples(math.NaN(), math.Inf(-1)), config.ReduceLast)
	assert.False(t, ok)
}

func TestReduceMatrix_DropsEmptySeries(t *testing.T) {
	matrix := model.Matrix{
		&model.SampleStream{Metric: model.Metric{"a": "1"}, Values: samples(1, 2, 3)},
		&model.SampleStream{Metric: model.Metric{"b": "2"}, Values: samples(math.NaN())},
	}
	vec := reduceMatrix(matrix, config.ReduceSum)
	require.Len(t, vec, 1) // all-NaN series dropped
	assert.Equal(t, 6.0, float64(vec[0].Value))
}
