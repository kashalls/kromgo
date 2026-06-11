package kromgo

import (
	"encoding/json"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapSeries(t *testing.T) {
	t.Parallel()
	mk := func(n int) model.Matrix {
		m := make(model.Matrix, n)
		for i := range m {
			m[i] = &model.SampleStream{}
		}
		return m
	}
	// Under the cap: returned unchanged.
	assert.Len(t, capSeries(mk(maxGraphSeries-1), slog.Default()), maxGraphSeries-1)
	// At the cap: unchanged.
	assert.Len(t, capSeries(mk(maxGraphSeries), slog.Default()), maxGraphSeries)
	// Over the cap: truncated to the cap.
	assert.Len(t, capSeries(mk(maxGraphSeries+25), slog.Default()), maxGraphSeries)
}

// A graph's range result routinely contains NaN/Inf samples (gaps, staleness,
// x/0). encoding/json errors on those, so a single one would 500 the whole JSON
// response; historyResponse must drop them, matching the chart's gap rendering.
func TestHistoryResponse_SkipsNonFinite(t *testing.T) {
	t.Parallel()
	matrix := model.Matrix{{
		Metric: model.Metric{"__name__": "up", "pod": "a"},
		Values: []model.SamplePair{
			{Timestamp: 1000, Value: 1.5},
			{Timestamp: 2000, Value: model.SampleValue(math.NaN())},
			{Timestamp: 3000, Value: model.SampleValue(math.Inf(1))},
			{Timestamp: 4000, Value: model.SampleValue(math.Inf(-1))},
			{Timestamp: 5000, Value: 2.5},
		},
	}}
	resp := historyResponse(
		&resolvedGraph{Graph: config.Graph{ID: "g", Title: "G"}},
		time.Unix(0, 0), time.Unix(10, 0), time.Minute, matrix,
	)

	require.Len(t, resp.Series, 1)
	data := resp.Series[0].Data
	require.Len(t, data, 2, "only the two finite samples survive")
	assert.Equal(t, 1.5, data[0].V)
	assert.Equal(t, int64(1), data[0].T, "ms timestamp is divided to seconds")
	assert.Equal(t, 2.5, data[1].V)

	// The payload must marshal — a surviving NaN/Inf would make json.Marshal fail.
	_, err := json.Marshal(resp)
	require.NoError(t, err)
}
