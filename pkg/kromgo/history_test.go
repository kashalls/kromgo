package kromgo

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
)

func makeRequest(params map[string]string) *http.Request {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	r := &http.Request{URL: &url.URL{RawQuery: q.Encode()}}
	return r
}

func TestParseHistoryParams_Defaults(t *testing.T) {
	before := time.Now()
	r := makeRequest(nil)
	start, end, step, err := parseHistoryParams(r)
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end.Before(before) || end.After(after) {
		t.Errorf("end not near now: %v", end)
	}
	if end.Sub(start) < 59*time.Minute || end.Sub(start) > 61*time.Minute {
		t.Errorf("default start not ~1h before end: diff=%v", end.Sub(start))
	}
	if step != time.Minute {
		// auto step for 1h window is 1h/100 = 36s, clamped to 1m
		t.Errorf("expected step=1m (clamped), got %v", step)
	}
}

func TestParseHistoryParams_RFC3339(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "2024-01-01T00:00:00Z",
		"end":   "2024-01-01T06:00:00Z",
	})
	start, end, _, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start.Unix() != 1704067200 {
		t.Errorf("wrong start: %v", start)
	}
	if end.Unix() != 1704088800 {
		t.Errorf("wrong end: %v", end)
	}
}

func TestParseHistoryParams_UnixTimestamp(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "1704067200",
		"end":   "1704088800",
	})
	start, end, _, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start.Unix() != 1704067200 {
		t.Errorf("wrong start: %v", start)
	}
	if end.Unix() != 1704088800 {
		t.Errorf("wrong end: %v", end)
	}
}

func TestParseHistoryParams_ExplicitStep(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "1704067200",
		"end":   "1704088800",
		"step":  "5m",
	})
	_, _, step, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step != 5*time.Minute {
		t.Errorf("expected 5m, got %v", step)
	}
}

func TestParseHistoryParams_StepClampedToMinute(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "1704067200",
		"end":   "1704088800",
		"step":  "10s",
	})
	_, _, step, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step != time.Minute {
		t.Errorf("expected step clamped to 1m, got %v", step)
	}
}

func TestParseHistoryParams_AutoStepLargerWindow(t *testing.T) {
	// 100h window: auto step = 100h/100 = 1h
	r := makeRequest(map[string]string{
		"start": "1704067200",
		"end":   "1704427200", // +100h
	})
	_, _, step, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step != time.Hour {
		t.Errorf("expected auto step=1h, got %v", step)
	}
}

func TestParseHistoryParams_Last(t *testing.T) {
	before := time.Now()
	r := makeRequest(map[string]string{"last": "7d"})
	start, end, _, err := parseHistoryParams(r)
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end.Before(before) || end.After(after) {
		t.Errorf("end not near now: %v", end)
	}
	if diff := end.Sub(start); diff < 7*24*time.Hour-time.Second || diff > 7*24*time.Hour+time.Second {
		t.Errorf("expected ~7d window, got %v", diff)
	}
}

func TestParseHistoryParams_LastOverridesStartEnd(t *testing.T) {
	r := makeRequest(map[string]string{
		"last":  "1d",
		"start": "1704067200",
		"end":   "1704088800",
	})
	_, _, _, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// last takes precedence — no start-after-end error despite conflicting params
}

func TestParseHistoryParams_LastInvalid(t *testing.T) {
	r := makeRequest(map[string]string{"last": "invalid"})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid last param")
	}
}

func TestParseHistoryParams_LastNegative(t *testing.T) {
	r := makeRequest(map[string]string{"last": "-1h"})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for negative last param")
	}
	if err != errNonPositiveDuration {
		t.Errorf("expected errNonPositiveDuration, got %v", err)
	}
}

func TestParseHistoryParams_LastZero(t *testing.T) {
	r := makeRequest(map[string]string{"last": "0"})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for zero last param")
	}
	if err != errNonPositiveDuration {
		t.Errorf("expected errNonPositiveDuration, got %v", err)
	}
}

func TestParseHistoryParams_StartAfterEnd(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "1704088800",
		"end":   "1704067200",
	})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for start > end")
	}
}

func TestParseHistoryParams_InvalidStart(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "not-a-time",
	})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid start")
	}
}

func TestParseHistoryParams_InvalidEnd(t *testing.T) {
	r := makeRequest(map[string]string{
		"end": "not-a-time",
	})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid end")
	}
}

func TestParseHistoryParams_InvalidStep(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "1704067200",
		"end":   "1704088800",
		"step":  "invalid",
	})
	_, _, _, err := parseHistoryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid step")
	}
}

func TestParseTimeParam_RFC3339(t *testing.T) {
	ts, err := parseTimeParam("2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Unix() != 1704067200 {
		t.Errorf("wrong timestamp: %v", ts.Unix())
	}
}

func TestParseTimeParam_Unix(t *testing.T) {
	ts, err := parseTimeParam("1704067200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Unix() != 1704067200 {
		t.Errorf("wrong timestamp: %v", ts.Unix())
	}
}

func TestParseTimeParam_Invalid(t *testing.T) {
	_, err := parseTimeParam("garbage")
	if err == nil {
		t.Fatal("expected error for invalid time param")
	}
}

func newHandler(historyCfg configuration.HistoryConfig) *KromgoHandler {
	return &KromgoHandler{
		Config: configuration.KromgoConfig{History: historyCfg},
	}
}

func TestHistoryEnabled_GlobalOff(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{Enabled: false})
	metric := configuration.Metric{Name: "test"}
	if h.historyEnabled(metric) {
		t.Error("expected history disabled by default")
	}
}

func TestHistoryEnabled_GlobalOn(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{Enabled: true})
	metric := configuration.Metric{Name: "test"}
	if !h.historyEnabled(metric) {
		t.Error("expected history enabled via global config")
	}
}

func TestHistoryEnabled_PerMetricOverrideOn(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{Enabled: false})
	metric := configuration.Metric{Name: "test", History: &configuration.MetricHistoryConfig{Enabled: boolPtr(true)}}
	if !h.historyEnabled(metric) {
		t.Error("expected per-metric history override to enable history")
	}
}

func TestHistoryEnabled_PerMetricOverrideOff(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{Enabled: true})
	metric := configuration.Metric{Name: "test", History: &configuration.MetricHistoryConfig{Enabled: boolPtr(false)}}
	if h.historyEnabled(metric) {
		t.Error("expected per-metric history override to disable history")
	}
}

func TestHistoryMaxDuration_Default(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{})
	metric := configuration.Metric{Name: "test"}
	if d := h.historyMaxDuration(metric); d != time.Hour {
		t.Errorf("expected default max duration 1h, got %v", d)
	}
}

func TestHistoryMaxDuration_GlobalConfigured(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{MaxDuration: "24h"})
	metric := configuration.Metric{Name: "test"}
	if d := h.historyMaxDuration(metric); d != 24*time.Hour {
		t.Errorf("expected 24h, got %v", d)
	}
}

func TestHistoryMaxDuration_PerMetricOverridesGlobal(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{MaxDuration: "24h"})
	metric := configuration.Metric{Name: "test", History: &configuration.MetricHistoryConfig{MaxDuration: "720h"}}
	if d := h.historyMaxDuration(metric); d != 720*time.Hour {
		t.Errorf("expected 720h, got %v", d)
	}
}

func TestHistoryMaxDuration_Unlimited(t *testing.T) {
	h := newHandler(configuration.HistoryConfig{MaxDuration: "0"})
	metric := configuration.Metric{Name: "test"}
	if d := h.historyMaxDuration(metric); d != 0 {
		t.Errorf("expected 0 (unlimited), got %v", d)
	}
}

func TestParseHistoryParams_StepDays(t *testing.T) {
	r := makeRequest(map[string]string{
		"start": "1704067200",
		"end":   "1704672000", // +7d
		"step":  "1d",
	})
	_, _, step, err := parseHistoryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if step != 24*time.Hour {
		t.Errorf("expected 24h step, got %v", step)
	}
}
