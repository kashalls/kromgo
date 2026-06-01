package kromgo

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRequest(params map[string]string) *http.Request {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return &http.Request{URL: &url.URL{RawQuery: q.Encode()}}
}

func TestParseHistoryParams_Valid(t *testing.T) {
	// Cases with absolute start/end. wantStart/wantEnd of 0 and wantStep of 0 mean
	// "don't assert this field".
	cases := []struct {
		name      string
		params    map[string]string
		wantStart int64
		wantEnd   int64
		wantStep  time.Duration
	}{
		{"rfc3339", map[string]string{"start": "2024-01-01T00:00:00Z", "end": "2024-01-01T06:00:00Z"}, 1704067200, 1704088800, 0},
		{"unix timestamp", map[string]string{"start": "1704067200", "end": "1704088800"}, 1704067200, 1704088800, 0},
		{"explicit step", map[string]string{"start": "1704067200", "end": "1704088800", "step": "5m"}, 0, 0, 5 * time.Minute},
		{"step clamped to minute", map[string]string{"start": "1704067200", "end": "1704088800", "step": "10s"}, 0, 0, time.Minute},
		{"auto step on larger window", map[string]string{"start": "1704067200", "end": "1704427200"}, 0, 0, time.Hour}, // 100h/100
		{"step in days", map[string]string{"start": "1704067200", "end": "1704672000", "step": "1d"}, 0, 0, 24 * time.Hour},
		{"last overrides start/end", map[string]string{"last": "1d", "start": "1704067200", "end": "1704088800"}, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, step, err := parseHistoryParams(makeRequest(tc.params))
			require.NoError(t, err)
			if tc.wantStart != 0 {
				assert.Equal(t, tc.wantStart, start.Unix())
			}
			if tc.wantEnd != 0 {
				assert.Equal(t, tc.wantEnd, end.Unix())
			}
			if tc.wantStep != 0 {
				assert.Equal(t, tc.wantStep, step)
			}
		})
	}
}

func TestParseHistoryParams_DefaultWindow(t *testing.T) {
	before := time.Now()
	start, end, step, err := parseHistoryParams(makeRequest(nil))
	require.NoError(t, err)
	assert.WithinDuration(t, before, end, time.Second, "end defaults to ~now")
	assert.WithinDuration(t, before.Add(-time.Hour), start, time.Second, "start defaults to ~1h before end")
	assert.Equal(t, time.Minute, step, "autoStep(1h) clamps to 1m")
}

func TestParseHistoryParams_Last(t *testing.T) {
	before := time.Now()
	start, end, _, err := parseHistoryParams(makeRequest(map[string]string{"last": "7d"}))
	require.NoError(t, err)
	assert.WithinDuration(t, before, end, time.Second)
	assert.WithinDuration(t, before.Add(-7*24*time.Hour), start, time.Second)
}

func TestParseHistoryParams_Errors(t *testing.T) {
	cases := []struct {
		name     string
		params   map[string]string
		sentinel error // nil means any error is acceptable
	}{
		{"invalid last", map[string]string{"last": "invalid"}, nil},
		{"negative last", map[string]string{"last": "-1h"}, errNonPositiveDuration},
		{"zero last", map[string]string{"last": "0"}, errNonPositiveDuration},
		{"start after end", map[string]string{"start": "1704088800", "end": "1704067200"}, errStartAfterEnd},
		{"invalid start", map[string]string{"start": "not-a-time"}, nil},
		{"invalid end", map[string]string{"end": "not-a-time"}, nil},
		{"invalid step", map[string]string{"start": "1704067200", "end": "1704088800", "step": "invalid"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseHistoryParams(makeRequest(tc.params))
			require.Error(t, err)
			if tc.sentinel != nil {
				assert.ErrorIs(t, err, tc.sentinel)
			}
		})
	}
}

func TestParseTimeParam(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantUnix int64
		wantErr  bool
	}{
		{"rfc3339", "2024-01-01T00:00:00Z", 1704067200, false},
		{"unix", "1704067200", 1704067200, false},
		{"invalid", "garbage", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts, err := parseTimeParam(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantUnix, ts.Unix())
		})
	}
}

func mustResolve(t *testing.T, m config.Metric, cfg config.KromgoConfig) *resolvedMetric {
	t.Helper()
	env, err := newCELEnv()
	require.NoError(t, err)
	rm, err := resolveMetric(m, cfg, env)
	require.NoError(t, err)
	return rm
}

// tsDefaults builds a config whose only setting is the default timeseries config.
func tsDefaults(rc config.TimeseriesConfig) config.KromgoConfig {
	return config.KromgoConfig{Defaults: config.Defaults{Timeseries: rc}}
}

func TestResolveMetric_HistoryEnabled(t *testing.T) {
	cases := []struct {
		name     string
		metric   config.Metric
		defaults config.TimeseriesConfig
		want     bool
	}{
		{"default off", config.Metric{Name: "test"}, config.TimeseriesConfig{Enabled: false}, false},
		{"default on", config.Metric{Name: "test"}, config.TimeseriesConfig{Enabled: true}, true},
		{"per-metric override on", config.Metric{Name: "test", Timeseries: &config.MetricTimeseriesConfig{Enabled: new(true)}}, config.TimeseriesConfig{Enabled: false}, true},
		{"per-metric override off", config.Metric{Name: "test", Timeseries: &config.MetricTimeseriesConfig{Enabled: new(false)}}, config.TimeseriesConfig{Enabled: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rm := mustResolve(t, tc.metric, tsDefaults(tc.defaults))
			assert.Equal(t, tc.want, rm.historyEnabled)
		})
	}
}

func TestResolveMetric_HistoryMax(t *testing.T) {
	cases := []struct {
		name   string
		metric config.Metric
		cfg    config.KromgoConfig
		want   time.Duration
	}{
		{"built-in default", config.Metric{Name: "test"}, config.KromgoConfig{}, time.Hour},
		{"default configured", config.Metric{Name: "test"}, tsDefaults(config.TimeseriesConfig{MaxDuration: "24h"}), 24 * time.Hour},
		{"per-metric overrides default", config.Metric{Name: "test", Timeseries: &config.MetricTimeseriesConfig{MaxDuration: "720h"}}, tsDefaults(config.TimeseriesConfig{MaxDuration: "24h"}), 720 * time.Hour},
		{"unlimited", config.Metric{Name: "test"}, tsDefaults(config.TimeseriesConfig{MaxDuration: "0"}), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rm := mustResolve(t, tc.metric, tc.cfg)
			assert.Equal(t, tc.want, rm.historyMax)
		})
	}
}

func TestResolveMetric_InvalidExprFailsFast(t *testing.T) {
	env, err := newCELEnv()
	require.NoError(t, err)
	cases := map[string]config.Metric{
		"syntax error":  {Name: "a", Value: "result +"},
		"not a string":  {Name: "b", Value: "result"},       // value must be string
		"unknown ident": {Name: "c", Color: "nope(result)"}, // bad color expr
	}
	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := resolveMetric(m, config.KromgoConfig{}, env)
			assert.Error(t, err)
		})
	}
}
