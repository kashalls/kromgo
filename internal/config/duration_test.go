package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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
	t.Parallel()
	cases := []struct {
		name    string
		cfg     KromgoConfig
		wantErr bool
	}{
		{
			"valid",
			KromgoConfig{
				Defaults: Defaults{Graph: GraphDefaults{MaxDuration: "24h"}},
				Graphs: []Graph{
					{ID: "cpu", Query: "q", MaxDuration: "7d"},
					{ID: "mem", Query: "q"},
				},
			},
			false,
		},
		{
			"invalid default maxDuration",
			KromgoConfig{Defaults: Defaults{Graph: GraphDefaults{MaxDuration: "bogus"}}},
			true,
		},
		{
			"invalid graph maxDuration",
			KromgoConfig{Graphs: []Graph{{ID: "cpu", Query: "q", MaxDuration: "not-a-duration"}}},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	badge := func(b Badge) KromgoConfig { return KromgoConfig{Badges: []Badge{b}} }
	cases := []struct {
		name    string
		cfg     KromgoConfig
		wantErr bool
	}{
		{"valid range", badge(Badge{ID: "ok", Query: "q", Type: TypeRange, Range: &RangeQuery{Last: "7d", Offset: "1d", Step: "1h", Reduce: ReduceAvg}}), false},
		{"type range without range block", badge(Badge{ID: "no-range", Query: "q", Type: TypeRange}), true},
		{"missing last", badge(Badge{ID: "no-last", Query: "q", Type: TypeRange, Range: &RangeQuery{Step: "1h"}}), true},
		{"unknown reducer", badge(Badge{ID: "bad-reduce", Query: "q", Type: TypeRange, Range: &RangeQuery{Last: "7d", Reduce: "median"}}), true},
		{"bad duration", badge(Badge{ID: "bad-dur", Query: "q", Type: TypeRange, Range: &RangeQuery{Last: "soon"}}), true},
		{"range block on instant", badge(Badge{ID: "range-on-instant", Query: "q", Range: &RangeQuery{Last: "7d"}}), true},
		{"unknown type", badge(Badge{ID: "bad-type", Query: "q", Type: "scalar"}), true},
		{"unknown style", badge(Badge{ID: "bad-style", Query: "q", Style: "fancy"}), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
