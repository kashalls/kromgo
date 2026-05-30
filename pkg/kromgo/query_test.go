package kromgo

import (
	"testing"
	"time"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/prometheus/common/model"
)

func samples(vals ...float64) []model.SamplePair {
	out := make([]model.SamplePair, len(vals))
	for i, v := range vals {
		out[i] = model.SamplePair{Timestamp: model.Time(i * 1000), Value: model.SampleValue(v)}
	}
	return out
}

func TestReduceSamples(t *testing.T) {
	values := samples(2, 5, 1, 4)
	cases := map[string]float64{
		"last":  4,
		"first": 2,
		"sum":   12,
		"avg":   3,
		"min":   1,
		"max":   5,
		"":      4, // defaults to last
	}
	for fn, want := range cases {
		if got := float64(reduceSamples(values, fn)); got != want {
			t.Errorf("reduce %q = %v, want %v", fn, got, want)
		}
	}
}

func TestReduceSamples_Empty(t *testing.T) {
	if got := reduceSamples(nil, "avg"); got != 0 {
		t.Errorf("reduce of empty series = %v, want 0", got)
	}
}

func TestRangeWindow_NoOffset(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	rc := &configuration.RangeConfig{Last: "7d", Step: "1h"}

	start, end, step := rangeWindow(rc, now)
	if !end.Equal(now) {
		t.Errorf("end = %v, want now %v", end, now)
	}
	if !start.Equal(now.Add(-7 * 24 * time.Hour)) {
		t.Errorf("start = %v, want now-7d", start)
	}
	if step != time.Hour {
		t.Errorf("step = %v, want 1h", step)
	}
}

func TestRangeWindow_WithOffset(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	rc := &configuration.RangeConfig{Last: "7d", Offset: "7d", Step: "1h"}

	start, end, _ := rangeWindow(rc, now)
	wantEnd := now.Add(-7 * 24 * time.Hour)    // 7d ago
	wantStart := now.Add(-14 * 24 * time.Hour) // 14d ago
	if !end.Equal(wantEnd) {
		t.Errorf("end = %v, want 7d ago %v", end, wantEnd)
	}
	if !start.Equal(wantStart) {
		t.Errorf("start = %v, want 14d ago %v", start, wantStart)
	}
}
