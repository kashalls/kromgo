package kromgo

import (
	"cmp"
	"fmt"
	"strconv"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/home-operations/kromgo/internal/config"
)

const (
	defaultGraphMaxDuration = time.Hour
	minRangeStep            = time.Minute
	defaultValueExpr        = "string(result)"
	defaultGraphWidth       = 600
	defaultGraphHeight      = 200
)

// resolvedBadge is a config.Badge with its per-request values resolved once at
// startup: the compiled value/color CEL programs, effective style, and (for type:
// range) the parsed range-query window. This keeps compilation off the request hot
// path and surfaces bad config at startup.
type resolvedBadge struct {
	config.Badge
	valueProg  cel.Program // compiled Value expression (always set)
	colorProg  cel.Program // compiled Color expression; nil when none
	style      string
	labelColor string      // resolved label-segment hex; "" = default grey (#555)
	iconPath   string      // resolved SVG path data for Icon; "" when none
	rangeQuery *rangeQuery // non-nil when Type == range
}

// resolvedGraph is a config.Graph with its window cap and default sparkline
// parameters resolved once at startup.
type resolvedGraph struct {
	config.Graph
	maxDuration time.Duration // 0 means unlimited
	defaults    chartParams   // request query params override these
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
	iconPath, err := resolveIcon(b.Icon)
	if err != nil {
		return nil, fmt.Errorf("badge %q: %w", b.ID, err)
	}

	// labelColor is a fixed color (not a CEL expression); "" leaves the renderer's
	// default grey. Resolve names/hex to hex now so the renderer just paints it.
	labelColor := cmp.Or(b.LabelColor, def.Badge.LabelColor)
	if labelColor != "" {
		labelColor = colorNameToHex(labelColor)
	}

	rb := &resolvedBadge{
		Badge:      b,
		style:      cmp.Or(b.Style, def.Badge.Style, config.StyleFlat),
		labelColor: labelColor,
		iconPath:   iconPath,
	}

	if rb.valueProg, err = compileStringExpr(env, b.ID, "value", cmp.Or(b.ValueExpr, defaultValueExpr)); err != nil {
		return nil, err
	}
	if b.ColorExpr != "" {
		if rb.colorProg, err = compileStringExpr(env, b.ID, "color", b.ColorExpr); err != nil {
			return nil, err
		}
	}
	if b.Type == config.TypeRange {
		if rb.rangeQuery, err = resolveRangeQuery(b); err != nil {
			return nil, err
		}
	}
	return rb, nil
}

// resolveGraph precomputes a graph's cache TTL, window cap, and default parameters.
func resolveGraph(g config.Graph, def config.Defaults, env *cel.Env) (*resolvedGraph, error) {
	theme := cmp.Or(g.Theme, def.Graph.Theme)
	if theme != "" && !validTheme(theme) {
		return nil, fmt.Errorf("graph %q: unknown theme %q", g.ID, theme)
	}

	font, err := resolveGraphFont(cmp.Or(g.Font, def.Graph.Font))
	if err != nil {
		return nil, fmt.Errorf("graph %q font: %w", g.ID, err)
	}

	rg := &resolvedGraph{
		Graph:       g,
		maxDuration: defaultGraphMaxDuration,
		defaults: chartParams{
			width:  cmp.Or(g.Width, def.Graph.Width, defaultGraphWidth),
			height: cmp.Or(g.Height, def.Graph.Height, defaultGraphHeight),
			legend: firstSet(true, g.Legend, def.Graph.Legend),
			fill:   firstSet(false, g.Fill, def.Graph.Fill),
			theme:  theme,
			title:  displayTitle(g.Title, g.ID),
			font:   font,
			format: formatSVG,
		},
	}
	if maxStr := cmp.Or(g.MaxDuration, def.Graph.MaxDuration); maxStr != "" {
		d, err := config.ParseDuration(maxStr)
		if err != nil {
			return nil, fmt.Errorf("graph %q maxDuration: %w", g.ID, err)
		}
		rg.maxDuration = d
	}
	if expr := cmp.Or(g.ValueExpr, def.Graph.ValueExpr); expr != "" {
		prog, err := compileStringExpr(env, g.ID, "value", expr)
		if err != nil {
			return nil, err
		}
		// Format each y-axis tick by evaluating the expression with result = the tick
		// value (an axis tick has no series, so no labels). A runtime eval error
		// degrades to a plain number rather than failing the whole render.
		rg.defaults.valueFormatter = func(f float64) string {
			s, err := evalStringExpr(prog, f, nil)
			if err != nil {
				return strconv.FormatFloat(f, 'f', -1, 64)
			}
			return s
		}
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

// firstSet returns the value of the first non-nil pointer, else fallback — the
// "optional override(s) with a default" pattern used across config resolution
// (a nil pointer means "unset", distinct from a zero value).
func firstSet[T any](fallback T, ptrs ...*T) T {
	for _, p := range ptrs {
		if p != nil {
			return *p
		}
	}
	return fallback
}
