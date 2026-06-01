package kromgo

import (
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/home-operations/kromgo/internal/config"
)

const (
	defaultHistoryMaxDuration = time.Hour
	minRangeStep              = time.Minute
	defaultValueExpr          = "string(result)"
)

// resolvedMetric is a config.Metric with its per-request values resolved once at
// startup: the compiled value/color CEL programs, effective timeseries settings,
// cache TTL, and (for type: range) the parsed range-query window. This keeps
// compilation/parsing off the request hot path and surfaces bad config at startup.
type resolvedMetric struct {
	config.Metric
	valueProg      cel.Program // compiled Value expression (always set)
	colorProg      cel.Program // compiled Color expression; nil when none
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
// its expressions, durations, or range query are invalid.
func resolveMetric(m config.Metric, cfg config.KromgoConfig, env *cel.Env) (*resolvedMetric, error) {
	rm := &resolvedMetric{
		Metric:         m,
		historyEnabled: cfg.Defaults.Timeseries.Enabled,
		historyMax:     defaultHistoryMaxDuration,
		cacheSeconds:   cfg.Defaults.CacheSeconds,
	}

	valueExpr := m.Value
	if valueExpr == "" {
		valueExpr = defaultValueExpr
	}
	var err error
	if rm.valueProg, err = compileStringExpr(env, m.Name, "value", valueExpr); err != nil {
		return nil, err
	}
	if m.Color != "" {
		if rm.colorProg, err = compileStringExpr(env, m.Name, "color", m.Color); err != nil {
			return nil, err
		}
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
		rq.step = autoStep(last)
	}

	return rq, nil
}

// autoStep picks a default range-query step: 1/100th of the window, clamped to minRangeStep.
func autoStep(window time.Duration) time.Duration {
	return max(window/100, minRangeStep)
}

// metricTitle returns the display title for a metric (its Title, falling back to Name).
func metricTitle(metric *resolvedMetric) string {
	if metric.Title != "" {
		return metric.Title
	}
	return metric.Name
}

// effectiveMaxDuration returns the metric's timeseries max-duration string (per-metric
// override, else default), or "" when neither is set (caller uses the built-in default).
func effectiveMaxDuration(m config.Metric, cfg config.KromgoConfig) string {
	if m.Timeseries != nil && m.Timeseries.MaxDuration != "" {
		return m.Timeseries.MaxDuration
	}
	return cfg.Defaults.Timeseries.MaxDuration
}
