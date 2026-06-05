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

func TestParseGraphParams_Valid(t *testing.T) {
	t.Parallel()
	// wantStart/wantEnd/wantStep of 0 mean "don't assert this field".
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
			t.Parallel()
			start, end, step, err := parseGraphParams(makeRequest(tc.params))
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

func TestParseGraphParams_DefaultWindow(t *testing.T) {
	t.Parallel()
	before := time.Now()
	start, end, step, err := parseGraphParams(makeRequest(nil))
	require.NoError(t, err)
	assert.WithinDuration(t, before, end, time.Second, "end defaults to ~now")
	assert.WithinDuration(t, before.Add(-time.Hour), start, time.Second, "start defaults to ~1h before end")
	assert.Equal(t, time.Minute, step, "autoStep(1h) clamps to 1m")
}

func TestParseGraphParams_Errors(t *testing.T) {
	t.Parallel()
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
		{"invalid step", map[string]string{"start": "1704067200", "end": "1704088800", "step": "invalid"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, err := parseGraphParams(makeRequest(tc.params))
			require.Error(t, err)
			if tc.sentinel != nil {
				assert.ErrorIs(t, err, tc.sentinel)
			}
		})
	}
}

func TestParseTimeParam(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
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

func TestResolveGraph_MaxDuration(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)
	cases := []struct {
		name  string
		graph config.Graph
		def   config.Defaults
		want  time.Duration
	}{
		{"built-in default", config.Graph{ID: "t"}, config.Defaults{}, time.Hour},
		{"default configured", config.Graph{ID: "t"}, config.Defaults{Graph: config.GraphDefaults{MaxDuration: "24h"}}, 24 * time.Hour},
		{"per-graph overrides default", config.Graph{ID: "t", MaxDuration: "720h"}, config.Defaults{Graph: config.GraphDefaults{MaxDuration: "24h"}}, 720 * time.Hour},
		{"unlimited", config.Graph{ID: "t", MaxDuration: "0"}, config.Defaults{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rg, err := resolveGraph(tc.graph, tc.def, env)
			require.NoError(t, err)
			assert.Equal(t, tc.want, rg.maxDuration)
		})
	}
}

func TestResolveGraph_InvalidTheme(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)
	_, err = resolveGraph(config.Graph{ID: "t", Query: "q", Theme: "nope"}, config.Defaults{}, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "theme")
}

func TestResolveGraph_DefaultParams(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)
	rg, err := resolveGraph(config.Graph{ID: "t"}, config.Defaults{}, env)
	require.NoError(t, err)
	assert.Equal(t, defaultGraphWidth, rg.defaults.width)
	assert.Equal(t, defaultGraphHeight, rg.defaults.height)
	assert.True(t, rg.defaults.legend, "legend defaults to true")
	assert.Nil(t, rg.defaults.valueFormatter, "no valueExpr ⇒ default numeric formatting")
}

func TestResolveGraph_ValueExpr(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)

	// A per-graph valueExpr compiles into a y-axis tick formatter.
	rg, err := resolveGraph(config.Graph{ID: "t", Query: "q", ValueExpr: `string(int(result)) + " pods"`}, config.Defaults{}, env)
	require.NoError(t, err)
	require.NotNil(t, rg.defaults.valueFormatter)
	assert.Equal(t, "42 pods", rg.defaults.valueFormatter(42.0))
	assert.Equal(t, "42 pods", rg.defaults.valueFormatter(42.7), "int() truncates the float tick")

	// defaults.graph.valueExpr applies when the graph doesn't set its own.
	rg, err = resolveGraph(config.Graph{ID: "t", Query: "q"}, config.Defaults{Graph: config.GraphDefaults{ValueExpr: "humanizeBytes(result)"}}, env)
	require.NoError(t, err)
	require.NotNil(t, rg.defaults.valueFormatter)
	assert.Equal(t, "1.5MB", rg.defaults.valueFormatter(1500000))

	// A per-graph valueExpr overrides the default.
	rg, err = resolveGraph(config.Graph{ID: "t", Query: "q", ValueExpr: "string(int(result))"}, config.Defaults{Graph: config.GraphDefaults{ValueExpr: "humanizeBytes(result)"}}, env)
	require.NoError(t, err)
	assert.Equal(t, "1500000", rg.defaults.valueFormatter(1500000))

	// A malformed expression fails at resolve (startup), not on a request.
	_, err = resolveGraph(config.Graph{ID: "t", Query: "q", ValueExpr: "nope("}, config.Defaults{}, env)
	require.Error(t, err)

	// An expression that compiles but returns a non-string is rejected too.
	_, err = resolveGraph(config.Graph{ID: "t", Query: "q", ValueExpr: "result + 1.0"}, config.Defaults{}, env)
	require.Error(t, err)
}

func TestResolveBadge_InvalidExprFailsFast(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)
	cases := map[string]config.Badge{
		"syntax error":  {ID: "a", Query: "q", ValueExpr: "result +"},
		"not a string":  {ID: "b", Query: "q", ValueExpr: "result"},       // value must be string
		"unknown ident": {ID: "c", Query: "q", ColorExpr: "nope(result)"}, // bad color expr
	}
	for name, b := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := resolveBadge(b, config.Defaults{}, env)
			assert.Error(t, err)
		})
	}
}

func TestResolveBadge_Style(t *testing.T) {
	t.Parallel()
	env, err := newCELEnv()
	require.NoError(t, err)

	rb, err := resolveBadge(config.Badge{ID: "a", Query: "q"}, config.Defaults{}, env)
	require.NoError(t, err)
	assert.Equal(t, config.StyleFlat, rb.style, "defaults to flat")

	rb, err = resolveBadge(config.Badge{ID: "a", Query: "q"}, config.Defaults{Badge: config.BadgeDefaults{Style: config.StylePlastic}}, env)
	require.NoError(t, err)
	assert.Equal(t, config.StylePlastic, rb.style, "inherits default style")

	rb, err = resolveBadge(config.Badge{ID: "a", Query: "q", Style: config.StyleFlatSquare}, config.Defaults{Badge: config.BadgeDefaults{Style: config.StylePlastic}}, env)
	require.NoError(t, err)
	assert.Equal(t, config.StyleFlatSquare, rb.style, "per-badge style wins")
}
