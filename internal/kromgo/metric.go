package kromgo

import (
	"cmp"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/home-operations/kromgo/internal/config"
)

const (
	defaultGraphMaxDuration = time.Hour
	minRangeStep            = time.Minute
	defaultValueExpr        = "string(result)"
	defaultGraphWidth       = 300
	defaultGraphHeight      = 80
	defaultGraphStroke      = 2.0
)

// resolvedBadge is a config.Badge with its per-request values resolved once at
// startup: the compiled value/color CEL programs, effective style and cache TTL,
// and (for type: range) the parsed range-query window. This keeps compilation off
// the request hot path and surfaces bad config at startup.
type resolvedBadge struct {
	config.Badge
	valueProg    cel.Program // compiled Value expression (always set)
	colorProg    cel.Program // compiled Color expression; nil when none
	cacheSeconds int
	style        string
	rangeQuery   *rangeQuery // non-nil when Type == range
}

// resolvedGraph is a config.Graph with its cache TTL, window cap, and default
// sparkline parameters resolved once at startup.
type resolvedGraph struct {
	config.Graph
	cacheSeconds int
	maxDuration  time.Duration // 0 means unlimited
	defaults     chartParams   // request query params override these
}

// rangeQuery is the resolved window for a type: range badge.
type rangeQuery struct {
	last   time.Duration
	offset time.Duration
	step   time.Duration
	reduce string
}

// resolveBadge precomputes a badge's request-time values.
func resolveBadge(b config.Badge, def config.Defaults, env *cel.Env) (*resolvedBadge, error) {
	rb := &resolvedBadge{
		Badge:        b,
		cacheSeconds: def.CacheSeconds,
		style:        cmp.Or(b.Style, def.Badge.Style, config.StyleFlat),
	}

	var err error
	if rb.valueProg, err = compileStringExpr(env, b.ID, "value", cmp.Or(b.Value, defaultValueExpr)); err != nil {
		return nil, err
	}
	if b.Color != "" {
		if rb.colorProg, err = compileStringExpr(env, b.ID, "color", b.Color); err != nil {
			return nil, err
		}
	}
	if b.CacheSeconds != nil {
		rb.cacheSeconds = *b.CacheSeconds
	}
	if b.Type == config.TypeRange {
		if rb.rangeQuery, err = resolveRangeQuery(b); err != nil {
			return nil, err
		}
	}
	return rb, nil
}

// resolveGraph precomputes a graph's cache TTL, window cap, and default parameters.
func resolveGraph(g config.Graph, def config.Defaults) (*resolvedGraph, error) {
	rg := &resolvedGraph{
		Graph:        g,
		cacheSeconds: def.CacheSeconds,
		maxDuration:  defaultGraphMaxDuration,
		defaults: chartParams{
			width:       cmp.Or(g.Width, def.Graph.Width, defaultGraphWidth),
			height:      cmp.Or(g.Height, def.Graph.Height, defaultGraphHeight),
			strokeWidth: cmp.Or(g.Stroke, def.Graph.Stroke, defaultGraphStroke),
			color:       g.Color,
			legend:      firstBool(true, g.Legend, def.Graph.Legend),
		},
	}
	if g.CacheSeconds != nil {
		rg.cacheSeconds = *g.CacheSeconds
	}
	if maxStr := cmp.Or(g.MaxDuration, def.Graph.MaxDuration); maxStr != "" {
		d, err := config.ParseDuration(maxStr)
		if err != nil {
			return nil, fmt.Errorf("graph %q maxDuration: %w", g.ID, err)
		}
		rg.maxDuration = d
	}
	return rg, nil
}

// resolveRangeQuery parses a range badge's windowed query (already validated by
// config.validate) into concrete durations, defaulting step to last/100 (min 1m).
func resolveRangeQuery(b config.Badge) (*rangeQuery, error) {
	last, err := config.ParseDuration(b.Range.Last)
	if err != nil {
		return nil, fmt.Errorf("badge %q range.last: %w", b.ID, err)
	}

	rq := &rangeQuery{last: last, reduce: cmp.Or(b.Range.Reduce, config.ReduceLast)}
	if b.Range.Offset != "" {
		if rq.offset, err = config.ParseDuration(b.Range.Offset); err != nil {
			return nil, fmt.Errorf("badge %q range.offset: %w", b.ID, err)
		}
	}
	if b.Range.Step != "" {
		if rq.step, err = config.ParseDuration(b.Range.Step); err != nil {
			return nil, fmt.Errorf("badge %q range.step: %w", b.ID, err)
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

// displayTitle returns title, falling back to id.
func displayTitle(title, id string) string {
	return cmp.Or(title, id)
}

// firstBool returns the first non-nil pointer's value, else fallback.
func firstBool(fallback bool, vs ...*bool) bool {
	for _, v := range vs {
		if v != nil {
			return *v
		}
	}
	return fallback
}
