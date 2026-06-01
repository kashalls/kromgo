package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"years", "1y", 365 * 24 * time.Hour, false},
		{"combined years and days", "1y30d", (365 + 30) * 24 * time.Hour, false},
		{"days and hours", "1d12h", 36 * time.Hour, false},
		{"minutes", "30m", 30 * time.Minute, false},
		{"hours", "6h", 6 * time.Hour, false},
		{"seconds", "90s", 90 * time.Second, false},
		{"milliseconds", "500ms", 500 * time.Millisecond, false},
		{"zero", "0", 0, false},
		{"empty", "", 0, true},
		{"invalid", "invalid", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDuration(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidate_Durations(t *testing.T) {
	cases := []struct {
		name    string
		cfg     KromgoConfig
		wantErr bool
	}{
		{
			"valid",
			KromgoConfig{
				Defaults: Defaults{Timeseries: TimeseriesConfig{MaxDuration: "24h"}},
				Metrics: []Metric{
					{Name: "cpu", Timeseries: &MetricTimeseriesConfig{MaxDuration: "7d"}},
					{Name: "mem"},
				},
			},
			false,
		},
		{
			"invalid default maxDuration",
			KromgoConfig{Defaults: Defaults{Timeseries: TimeseriesConfig{MaxDuration: "bogus"}}},
			true,
		},
		{
			"invalid metric maxDuration",
			KromgoConfig{Metrics: []Metric{{Name: "cpu", Timeseries: &MetricTimeseriesConfig{MaxDuration: "not-a-duration"}}}},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_RangeType(t *testing.T) {
	cases := []struct {
		name    string
		cfg     KromgoConfig
		wantErr bool
	}{
		{"valid range", KromgoConfig{Metrics: []Metric{{Name: "ok", Type: TypeRange, Range: &RangeQuery{Last: "7d", Offset: "1d", Step: "1h", Reduce: ReduceAvg}}}}, false},
		{"type range without range block", KromgoConfig{Metrics: []Metric{{Name: "no-range", Type: TypeRange}}}, true},
		{"missing last", KromgoConfig{Metrics: []Metric{{Name: "no-last", Type: TypeRange, Range: &RangeQuery{Step: "1h"}}}}, true},
		{"unknown reducer", KromgoConfig{Metrics: []Metric{{Name: "bad-reduce", Type: TypeRange, Range: &RangeQuery{Last: "7d", Reduce: "median"}}}}, true},
		{"bad duration", KromgoConfig{Metrics: []Metric{{Name: "bad-dur", Type: TypeRange, Range: &RangeQuery{Last: "soon"}}}}, true},
		{"range block on instant", KromgoConfig{Metrics: []Metric{{Name: "range-on-instant", Range: &RangeQuery{Last: "7d"}}}}, true},
		{"unknown type", KromgoConfig{Metrics: []Metric{{Name: "bad-type", Type: "scalar"}}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
