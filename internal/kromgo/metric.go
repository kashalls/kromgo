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
		historyEnabled: cfg.History.Enabled,
		historyMax:     defaultHistoryMaxDuration,
		cacheSeconds:   cfg.CacheSeconds,
	}

	if m.ValueTemplate != "" {
		str := m.ValueTemplate
		if resolved, ok := cfg.Templates[str]; ok {
			str = resolved
		}
		tmpl, err := template.New(m.Name).Funcs(templateFuncs).Parse(str)
		if err != nil {
			return nil, fmt.Errorf("metric %q valueTemplate: %w", m.Name, err)
		}
		rm.template = tmpl
	}

	if m.History != nil && m.History.Enabled != nil {
		rm.historyEnabled = *m.History.Enabled
	}

	if maxStr := effectiveMaxDuration(m, cfg); maxStr != "" {
		d, err := config.ParseDuration(maxStr)
		if err != nil {
			return nil, fmt.Errorf("metric %q history.maxDuration: %w", m.Name, err)
		}
		rm.historyMax = d
	}

	if m.CacheSeconds != nil {
		rm.cacheSeconds = *m.CacheSeconds
	}

	return rm, nil
}

// effectiveMaxDuration returns the metric's history max-duration string (per-metric
// override, else global), or "" when neither is set (caller uses the default).
func effectiveMaxDuration(m config.Metric, cfg config.KromgoConfig) string {
	if m.History != nil && m.History.MaxDuration != "" {
		return m.History.MaxDuration
	}
	return cfg.History.MaxDuration
}
