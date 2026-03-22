package configuration

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

func TestValidateHistoryDurations_Valid(t *testing.T) {
	config := KromgoConfig{
		History: HistoryConfig{MaxDuration: "24h"},
		Metrics: []Metric{
			{Name: "cpu", History: &MetricHistoryConfig{MaxDuration: "7d"}},
			{Name: "mem"},
		},
	}
	if err := validateHistoryDurations(config); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHistoryDurations_InvalidGlobal(t *testing.T) {
	config := KromgoConfig{
		History: HistoryConfig{MaxDuration: "bogus"},
	}
	if err := validateHistoryDurations(config); err == nil {
		t.Fatal("expected error for invalid global maxDuration")
	}
}

func TestValidateHistoryDurations_InvalidMetric(t *testing.T) {
	config := KromgoConfig{
		Metrics: []Metric{
			{Name: "cpu", History: &MetricHistoryConfig{MaxDuration: "not-a-duration"}},
		},
	}
	if err := validateHistoryDurations(config); err == nil {
		t.Fatal("expected error for invalid metric maxDuration")
	}
}
