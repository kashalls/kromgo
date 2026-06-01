// Package config defines kromgo's configuration model and loads it from YAML and
// the environment.
package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// DefaultPath is the config file location used when none is provided.
const DefaultPath = "/kromgo/config.yaml"

// KromgoConfig is the top-level YAML configuration. Endpoints are split by output
// type: badges render an instant value (SVG / shields.io JSON / kromgo JSON) and
// graphs render a time series (SVG sparkline / history JSON).
type KromgoConfig struct {
	Prometheus string   `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Defaults   Defaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Badges     []Badge  `yaml:"badges,omitempty" json:"badges,omitempty"`
	Graphs     []Graph  `yaml:"graphs,omitempty" json:"graphs,omitempty"`
}

// Defaults holds values applied to every endpoint, each overridable per endpoint.
type Defaults struct {
	// Hidden is the default index-page visibility. Defaults to true (all hidden).
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// CacheSeconds is the default Cache-Control max-age (in seconds). 0 disables caching.
	CacheSeconds int `yaml:"cacheSeconds,omitempty" json:"cacheSeconds,omitempty"`
	// Badge holds the default badge rendering settings.
	Badge BadgeDefaults `yaml:"badge,omitempty" json:"badge,omitempty"`
	// Graph holds the default graph rendering settings.
	Graph GraphDefaults `yaml:"graph,omitempty" json:"graph,omitempty"`
}

// BadgeDefaults holds the default SVG badge rendering settings.
type BadgeDefaults struct {
	// Font is an optional path to a TrueType font. Empty uses an embedded default font.
	Font string `yaml:"font,omitempty" json:"font,omitempty"`
	// Size is the font size in points (defaults to 11).
	Size int `yaml:"size,omitempty" json:"size,omitempty"`
	// Style is the default badge style: flat (default), flat-square, or plastic.
	Style string `yaml:"style,omitempty" json:"style,omitempty"`
}

// GraphDefaults holds the default graph rendering settings.
type GraphDefaults struct {
	// MaxDuration caps the requested time window (e.g. "24h", "7d"). Defaults to "1h"; "0" is unlimited.
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty"`
	// Width is the image width in pixels (defaults to 600).
	Width int `yaml:"width,omitempty" json:"width,omitempty"`
	// Height is the image height in pixels (defaults to 200).
	Height int `yaml:"height,omitempty" json:"height,omitempty"`
	// Legend toggles the series legend (defaults to true).
	Legend *bool `yaml:"legend,omitempty" json:"legend,omitempty"`
	// Theme selects the color theme (e.g. "dark", "grafana", "catppuccin-mocha", "dracula").
	Theme string `yaml:"theme,omitempty" json:"theme,omitempty"`
}

// Badge defines an instant-value endpoint at /badges/{id}.
type Badge struct {
	// ID is the URL path segment: /badges/{id}.
	ID string `yaml:"id" json:"id"`
	// Title is the display label (defaults to ID).
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Query is the PromQL expression to run.
	Query string `yaml:"query" json:"query"`
	// Type selects how the value is computed: "instant" (default) or "range" (reduce a window).
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// Range configures the windowed range query when Type is "range".
	Range *RangeQuery `yaml:"range,omitempty" json:"range,omitempty"`
	// Value is a CEL expression producing the displayed string. It receives `result`
	// (the sample value, double) and `labels` (map). Defaults to string(result).
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
	// Color is a CEL expression producing the color name or hex. Empty means no color.
	Color string `yaml:"color,omitempty" json:"color,omitempty"`
	// Style overrides defaults.badge.style for this badge.
	Style string `yaml:"style,omitempty" json:"style,omitempty"`
	// Hidden overrides defaults.hidden for this badge.
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// CacheSeconds overrides defaults.cacheSeconds for this badge.
	CacheSeconds *int `yaml:"cacheSeconds,omitempty" json:"cacheSeconds,omitempty"`
}

// Graph defines a time-series endpoint at /graphs/{id}.
type Graph struct {
	// ID is the URL path segment: /graphs/{id}.
	ID string `yaml:"id" json:"id"`
	// Title is the display label (defaults to ID).
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Query is the PromQL expression to run as a range query.
	Query string `yaml:"query" json:"query"`
	// MaxDuration overrides defaults.graph.maxDuration for this graph.
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty"`
	// Width overrides defaults.graph.width for this graph.
	Width int `yaml:"width,omitempty" json:"width,omitempty"`
	// Height overrides defaults.graph.height for this graph.
	Height int `yaml:"height,omitempty" json:"height,omitempty"`
	// Legend overrides defaults.graph.legend for this graph.
	Legend *bool `yaml:"legend,omitempty" json:"legend,omitempty"`
	// Theme overrides defaults.graph.theme for this graph.
	Theme string `yaml:"theme,omitempty" json:"theme,omitempty"`
	// Hidden overrides defaults.hidden for this graph.
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// CacheSeconds overrides defaults.cacheSeconds for this graph.
	CacheSeconds *int `yaml:"cacheSeconds,omitempty" json:"cacheSeconds,omitempty"`
}

// RangeQuery configures a windowed range query (Badge.Type == "range"). The window
// is end = now - offset, start = end - last; each series is reduced to one value.
type RangeQuery struct {
	// Last is the window length (e.g. "7d"). Required.
	Last string `yaml:"last" json:"last"`
	// Offset shifts the window back in time (e.g. "7d"). Defaults to none (window ends now).
	Offset string `yaml:"offset,omitempty" json:"offset,omitempty"`
	// Step is the range-query resolution (e.g. "1h"). Defaults to last/100, min 1m.
	Step string `yaml:"step,omitempty" json:"step,omitempty"`
	// Reduce collapses each series to one value: last (default), first, avg, min, max, sum.
	Reduce string `yaml:"reduce,omitempty" json:"reduce,omitempty"`
}

// Query type, range-reduce, and badge-style values.
const (
	TypeInstant = "instant"
	TypeRange   = "range"

	ReduceLast  = "last"
	ReduceFirst = "first"
	ReduceAvg   = "avg"
	ReduceMin   = "min"
	ReduceMax   = "max"
	ReduceSum   = "sum"

	StyleFlat       = "flat"
	StyleFlatSquare = "flat-square"
	StylePlastic    = "plastic"
)

// ValidReduce is the set of supported range-query reducers.
var ValidReduce = map[string]bool{
	ReduceLast: true, ReduceFirst: true, ReduceAvg: true,
	ReduceMin: true, ReduceMax: true, ReduceSum: true,
}

// ValidStyle is the set of supported badge styles.
var ValidStyle = map[string]bool{
	StyleFlat: true, StyleFlatSquare: true, StylePlastic: true,
}

// legacyKeys are top-level keys from the pre-0.12 schema; their presence triggers a
// pointed migration error rather than a generic "unknown field".
var legacyKeys = []string{"metrics", "badge", "hideAll", "history", "templates"}

// Load reads, parses, and validates the config file at path. An empty path uses DefaultPath.
func Load(path string) (KromgoConfig, error) {
	if path == "" {
		path = DefaultPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return KromgoConfig{}, fmt.Errorf("reading config file: %w", err)
	}

	if err := checkLegacy(data); err != nil {
		return KromgoConfig{}, err
	}

	// Strict decoding: an unknown/typo'd or stale key is an error, not a silent no-op.
	var cfg KromgoConfig
	if err := yaml.Load(data, &cfg, yaml.WithKnownFields()); err != nil {
		return KromgoConfig{}, fmt.Errorf("parsing config yaml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return KromgoConfig{}, err
	}

	return cfg, nil
}

// checkLegacy detects a pre-0.12 config and points at the migration guide.
func checkLegacy(data []byte) error {
	var probe map[string]any
	if yaml.Load(data, &probe) != nil {
		return nil // malformed YAML; let strict decoding report it
	}
	for _, k := range legacyKeys {
		if _, ok := probe[k]; ok {
			return fmt.Errorf("config key %q is from the pre-0.12 schema; endpoints are now split into `badges:` and `graphs:` sections — see \"Upgrading 0.11 → 0.12\" in the README", k)
		}
	}
	return nil
}

// validate checks defaults, then every badge and graph.
func (c KromgoConfig) validate() error {
	if s := c.Defaults.Graph.MaxDuration; s != "" {
		if _, err := ParseDuration(s); err != nil {
			return fmt.Errorf("defaults.graph.maxDuration: %w", err)
		}
	}
	if s := c.Defaults.Badge.Style; s != "" && !ValidStyle[s] {
		return fmt.Errorf("defaults.badge.style: unknown style %q", s)
	}

	seen := map[string]bool{}
	for _, b := range c.Badges {
		if err := b.validate(); err != nil {
			return err
		}
		if seen[b.ID] {
			return fmt.Errorf("badge %q: duplicate id", b.ID)
		}
		seen[b.ID] = true
	}

	seen = map[string]bool{}
	for _, g := range c.Graphs {
		if err := g.validate(); err != nil {
			return err
		}
		if seen[g.ID] {
			return fmt.Errorf("graph %q: duplicate id", g.ID)
		}
		seen[g.ID] = true
	}
	return nil
}

// validate checks a badge's id, query, style, and range-query block.
func (b Badge) validate() error {
	if b.ID == "" || b.Query == "" {
		return fmt.Errorf("badge %q: id and query are required", b.ID)
	}
	if b.Style != "" && !ValidStyle[b.Style] {
		return fmt.Errorf("badge %q: unknown style %q", b.ID, b.Style)
	}
	switch b.Type {
	case "", TypeInstant:
		if b.Range != nil {
			return fmt.Errorf("badge %q: range block is only valid with type: range", b.ID)
		}
	case TypeRange:
		if b.Range == nil || b.Range.Last == "" {
			return fmt.Errorf("badge %q: type range requires range.last", b.ID)
		}
		for name, val := range map[string]string{"last": b.Range.Last, "offset": b.Range.Offset, "step": b.Range.Step} {
			if val == "" {
				continue
			}
			if _, err := ParseDuration(val); err != nil {
				return fmt.Errorf("badge %q range.%s: %w", b.ID, name, err)
			}
		}
		if b.Range.Reduce != "" && !ValidReduce[b.Range.Reduce] {
			return fmt.Errorf("badge %q range.reduce: unknown reducer %q", b.ID, b.Range.Reduce)
		}
	default:
		return fmt.Errorf("badge %q: unknown type %q (want %q or %q)", b.ID, b.Type, TypeInstant, TypeRange)
	}
	return nil
}

// validate checks a graph's id, query, and maxDuration.
func (g Graph) validate() error {
	if g.ID == "" || g.Query == "" {
		return fmt.Errorf("graph %q: id and query are required", g.ID)
	}
	if g.MaxDuration != "" {
		if _, err := ParseDuration(g.MaxDuration); err != nil {
			return fmt.Errorf("graph %q maxDuration: %w", g.ID, err)
		}
	}
	return nil
}
