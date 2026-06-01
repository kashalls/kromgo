package config

import (
	"testing"
	"time"
)

func TestParseDuration_Days(t *testing.T) {
	d, err := ParseDuration("7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 7*24*time.Hour {
		t.Errorf("expected 168h, got %v", d)
	}
}

func TestParseDuration_Years(t *testing.T) {
	d, err := ParseDuration("1y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 365*24*time.Hour {
		t.Errorf("expected 8760h, got %v", d)
	}
}

func TestParseDuration_Combined(t *testing.T) {
	d, err := ParseDuration("1y30d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != (365+30)*24*time.Hour {
		t.Errorf("expected %v, got %v", (365+30)*24*time.Hour, d)
	}
}

func TestParseDuration_DaysAndHours(t *testing.T) {
	d, err := ParseDuration("1d12h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 36*time.Hour {
		t.Errorf("expected 36h, got %v", d)
	}
}

func TestParseDuration_StandardUnits(t *testing.T) {
	cases := map[string]time.Duration{
		"30m":   30 * time.Minute,
		"6h":    6 * time.Hour,
		"90s":   90 * time.Second,
		"500ms": 500 * time.Millisecond,
	}
	for s, want := range cases {
		d, err := ParseDuration(s)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", s, err)
			continue
		}
		if d != want {
			t.Errorf("%s: expected %v, got %v", s, want, d)
		}
	}
}

func TestParseDuration_Zero(t *testing.T) {
	d, err := ParseDuration("0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	_, err := ParseDuration("invalid")
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := KromgoConfig{
		Defaults: Defaults{Timeseries: TimeseriesConfig{MaxDuration: "24h"}},
		Metrics: []Metric{
			{Name: "cpu", Timeseries: &MetricTimeseriesConfig{MaxDuration: "7d"}},
			{Name: "mem"},
		},
	}
	if err := cfg.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidDefault(t *testing.T) {
	cfg := KromgoConfig{
		Defaults: Defaults{Timeseries: TimeseriesConfig{MaxDuration: "bogus"}},
	}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid defaults.timeseries.maxDuration")
	}
}

func TestValidate_InvalidMetric(t *testing.T) {
	cfg := KromgoConfig{
		Metrics: []Metric{
			{Name: "cpu", Timeseries: &MetricTimeseriesConfig{MaxDuration: "not-a-duration"}},
		},
	}
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for invalid metric maxDuration")
	}
}

func TestValidate_RangeType(t *testing.T) {
	ok := KromgoConfig{Metrics: []Metric{
		{Name: "ok", Type: TypeRange, Range: &RangeQuery{Last: "7d", Offset: "1d", Step: "1h", Reduce: ReduceAvg}},
	}}
	if err := ok.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bad := []KromgoConfig{
		{Metrics: []Metric{{Name: "no-range", Type: TypeRange}}},                                // type range without range block
		{Metrics: []Metric{{Name: "no-last", Type: TypeRange, Range: &RangeQuery{Step: "1h"}}}}, // missing last
		{Metrics: []Metric{{Name: "bad-reduce", Type: TypeRange, Range: &RangeQuery{Last: "7d", Reduce: "median"}}}},
		{Metrics: []Metric{{Name: "bad-dur", Type: TypeRange, Range: &RangeQuery{Last: "soon"}}}},
		{Metrics: []Metric{{Name: "range-on-instant", Range: &RangeQuery{Last: "7d"}}}}, // range block without type: range
		{Metrics: []Metric{{Name: "bad-type", Type: "scalar"}}},
	}
	for _, cfg := range bad {
		if err := cfg.validate(); err == nil {
			t.Errorf("expected validation error for %q", cfg.Metrics[0].Name)
		}
	}
}
