package kromgo

import (
	"fmt"
	"text/template"
	"time"

	"github.com/home-operations/kromgo/internal/config"
)

const (
	defaultHistoryMaxDuration = time.Hour
	minRangeStep              = time.Minute
)

// resolvedMetric is a config.Metric with its per-request values resolved once at
// startup: the compiled value template, effective timeseries settings, cache TTL,
// and (for type: range) the parsed range-query window. This keeps parsing off the
// request hot path and surfaces malformed config at startup.
type resolvedMetric struct {
	config.Metric
	template       *template.Template // compiled ValueTemplate; nil when none
	historyEnabled bool
	historyMax     time.Duration // 0 means unlimited
	cacheSeconds   int
	rangeQuery     *rangeQuery // non-nil when Type == range
}

// rangeQuery is the resolved window for a type: range metric.
type rangeQuery struct {
	last   time.Duration
	offset time.Duration
	step   time.Duration
	reduce string
}

// resolveMetric precomputes a metric's request-time values, returning an error if
// its value template, durations, or range query are invalid.
func resolveMetric(m config.Metric, cfg config.KromgoConfig) (*resolvedMetric, error) {
	rm := &resolvedMetric{
		Metric:         m,
		historyEnabled: cfg.Defaults.Timeseries.Enabled,
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

	if m.Timeseries != nil && m.Timeseries.Enabled != nil {
		rm.historyEnabled = *m.Timeseries.Enabled
	}

	if maxStr := effectiveMaxDuration(m, cfg); maxStr != "" {
		d, err := config.ParseDuration(maxStr)
		if err != nil {
			return nil, fmt.Errorf("metric %q timeseries.maxDuration: %w", m.Name, err)
		}
		rm.historyMax = d
	}

	if m.CacheSeconds != nil {
		rm.cacheSeconds = *m.CacheSeconds
	}

	if m.Type == config.TypeRange {
		rq, err := resolveRangeQuery(m)
		if err != nil {
			return nil, err
		}
		rm.rangeQuery = rq
	}

	return rm, nil
}

// resolveRangeQuery parses the windowed range-query config (already validated by
// config.validate) into concrete durations, defaulting step to last/100 (min 1m).
func resolveRangeQuery(m config.Metric) (*rangeQuery, error) {
	last, err := config.ParseDuration(m.Range.Last)
	if err != nil {
		return nil, fmt.Errorf("metric %q range.last: %w", m.Name, err)
	}

	rq := &rangeQuery{last: last, reduce: m.Range.Reduce}
	if rq.reduce == "" {
		rq.reduce = config.ReduceLast
	}

	if m.Range.Offset != "" {
		if rq.offset, err = config.ParseDuration(m.Range.Offset); err != nil {
			return nil, fmt.Errorf("metric %q range.offset: %w", m.Name, err)
		}
	}

	if m.Range.Step != "" {
		if rq.step, err = config.ParseDuration(m.Range.Step); err != nil {
			return nil, fmt.Errorf("metric %q range.step: %w", m.Name, err)
		}
	} else {
		rq.step = max(last/100, minRangeStep)
	}

	return rq, nil
}

// effectiveMaxDuration returns the metric's timeseries max-duration string (per-metric
// override, else default), or "" when neither is set (caller uses the built-in default).
func effectiveMaxDuration(m config.Metric, cfg config.KromgoConfig) string {
	if m.Timeseries != nil && m.Timeseries.MaxDuration != "" {
		return m.Timeseries.MaxDuration
	}
	return cfg.Defaults.Timeseries.MaxDuration
}
