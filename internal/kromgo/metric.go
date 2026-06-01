package kromgo

import (
	"fmt"
	"text/template"
	"time"

	"github.com/home-operations/kromgo/internal/config"
)

const defaultHistoryMaxDuration = time.Hour

// resolvedMetric is a config.Metric with its per-request values resolved once at
// startup: the compiled value template, effective history settings, and cache TTL.
// This keeps template parsing and duration parsing off the request hot path and
// surfaces malformed templates/durations at startup instead of per request.
type resolvedMetric struct {
	config.Metric
	template       *template.Template // compiled ValueTemplate; nil when none
	historyEnabled bool
	historyMax     time.Duration // 0 means unlimited
	cacheSeconds   int
}

// resolveMetric precomputes a metric's request-time values, returning an error if
// its value template or history duration is invalid.
func resolveMetric(m config.Metric, cfg config.KromgoConfig) (*resolvedMetric, error) {
	rm := &resolvedMetric{
		Metric:         m,
		historyEnabled: cfg.Defaults.Range.Enabled,
		historyMax:     defaultHistoryMaxDuration,
		cacheSeconds:   cfg.Defaults.CacheSeconds,
	}

	if m.ValueTemplate != "" {
		tmpl, err := template.New(m.Name).Funcs(templateFuncs).Parse(m.ValueTemplate)
		if err != nil {
			return nil, fmt.Errorf("metric %q valueTemplate: %w", m.Name, err)
		}
		rm.template = tmpl
	}

	if m.Range != nil && m.Range.Enabled != nil {
		rm.historyEnabled = *m.Range.Enabled
	}

	if maxStr := effectiveMaxDuration(m, cfg); maxStr != "" {
		d, err := config.ParseDuration(maxStr)
		if err != nil {
			return nil, fmt.Errorf("metric %q range.maxDuration: %w", m.Name, err)
		}
		rm.historyMax = d
	}

	if m.CacheSeconds != nil {
		rm.cacheSeconds = *m.CacheSeconds
	}

	return rm, nil
}

// effectiveMaxDuration returns the metric's range max-duration string (per-metric
// override, else default), or "" when neither is set (caller uses the built-in default).
func effectiveMaxDuration(m config.Metric, cfg config.KromgoConfig) string {
	if m.Range != nil && m.Range.MaxDuration != "" {
		return m.Range.MaxDuration
	}
	return cfg.Defaults.Range.MaxDuration
}
