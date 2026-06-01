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

// KromgoConfig is the top-level YAML configuration.
type KromgoConfig struct {
	Prometheus string   `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Metrics    []Metric `yaml:"metrics" json:"metrics"`
	Badge      Badge    `yaml:"badge,omitempty" json:"badge,omitempty"`
	// Defaults holds the default values for the per-metric fields that support it.
	Defaults Defaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

// Defaults holds the default values applied to every metric, each overridable by
// the same-named field on an individual Metric.
type Defaults struct {
	// Hidden is the default index-page visibility. Defaults to true (all hidden).
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// CacheSeconds is the default Cache-Control max-age (in seconds). 0 disables caching.
	CacheSeconds int `yaml:"cacheSeconds,omitempty" json:"cacheSeconds,omitempty"`
	// Timeseries controls access to the format=history and format=chart output formats.
	Timeseries TimeseriesConfig `yaml:"timeseries,omitempty" json:"timeseries,omitempty"`
}

// TimeseriesConfig holds the default settings for the format=history and format=chart
// time-series output endpoints.
type TimeseriesConfig struct {
	// Enabled must be true to allow format=history and format=chart requests. Defaults to false.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// MaxDuration caps the time window per request (e.g. "24h", "7d"). Defaults to "1h".
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty"`
}

// Metric defines one queryable endpoint at /{name}.
type Metric struct {
	// Name is the URL path segment used to query the metric.
	Name string `yaml:"name" json:"name"`
	// Title is the display label in badge/endpoint responses (defaults to Name).
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Query is the PromQL expression to run.
	Query string `yaml:"query" json:"query"`
	// Type selects how the value is computed: "instant" (default) runs an instant
	// query; "range" runs a range query over Range's window and reduces it to a value.
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	// Range configures the windowed range query when Type is "range".
	Range *RangeQuery `yaml:"range,omitempty" json:"range,omitempty"`
	// Label extracts the value from this query-result label instead of the sample value.
	Label string `yaml:"label,omitempty" json:"label,omitempty"`
	// Prefix is prepended to the value in the response.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	// Suffix is appended to the value in the response.
	Suffix string `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	// ValueTemplate is an inline Go template applied to the value before prefix/suffix
	// are added. Available functions: simplifyDays, humanBytes, humanSIBytes,
	// humanDuration, humanizeThousands, toUpper, toLower, trim. Use a YAML anchor to
	// reuse one across metrics. Example: "{{ . | simplifyDays }}".
	ValueTemplate string `yaml:"valueTemplate,omitempty" json:"valueTemplate,omitempty"`
	// Colors assigns a response color based on the numeric value.
	Colors []MetricColor `yaml:"colors,omitempty" json:"colors,omitempty"`
	// Hidden overrides defaults.hidden for this metric. If nil, the default applies.
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// Timeseries overrides defaults.timeseries for this metric. If nil, the defaults apply.
	Timeseries *MetricTimeseriesConfig `yaml:"timeseries,omitempty" json:"timeseries,omitempty"`
	// CacheSeconds overrides defaults.cacheSeconds for this metric. If nil, the default applies.
	CacheSeconds *int `yaml:"cacheSeconds,omitempty" json:"cacheSeconds,omitempty"`
}

// RangeQuery configures a windowed range query (Metric.Type == "range"). The window
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

// MetricTimeseriesConfig overrides the default TimeseriesConfig for a single metric.
type MetricTimeseriesConfig struct {
	// Enabled overrides defaults.timeseries.enabled for this metric.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	// MaxDuration overrides defaults.timeseries.maxDuration for this metric (e.g. "24h").
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty"`
}

// MetricColor maps a numeric range to a color and an optional display override.
type MetricColor struct {
	Min           float64 `yaml:"min" json:"min"`
	Max           float64 `yaml:"max" json:"max"`
	Color         string  `yaml:"color,omitempty" json:"color,omitempty"`
	ValueOverride string  `yaml:"valueOverride,omitempty" json:"valueOverride,omitempty"`
}

// Badge configures SVG badge rendering.
type Badge struct {
	// Font is an optional path to a TrueType font. When empty, an embedded default font is used.
	Font string `yaml:"font,omitempty" json:"font,omitempty"`
	// Size is the font size in points (defaults to 11).
	Size int `yaml:"size,omitempty" json:"size,omitempty"`
}

// Load reads, parses, and validates the config file at path. An empty path uses DefaultPath.
func Load(path string) (KromgoConfig, error) {
	if path == "" {
		path = DefaultPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return KromgoConfig{}, fmt.Errorf("reading config file: %w", err)
	}

	// Strict decoding: an unknown/typo'd or stale key (e.g. the old hideAll/history)
	// is an error, not a silent no-op.
	var cfg KromgoConfig
	if err := yaml.Load(data, &cfg, yaml.WithKnownFields()); err != nil {
		return KromgoConfig{}, fmt.Errorf("parsing config yaml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return KromgoConfig{}, err
	}

	return cfg, nil
}

// MetricsByName indexes the configured metrics by their Name for O(1) lookup.
func (c KromgoConfig) MetricsByName() map[string]Metric {
	out := make(map[string]Metric, len(c.Metrics))
	for _, m := range c.Metrics {
		out[m.Name] = m
	}
	return out
}

// Query type and range-reduce values.
const (
	TypeInstant = "instant"
	TypeRange   = "range"

	ReduceLast  = "last"
	ReduceFirst = "first"
	ReduceAvg   = "avg"
	ReduceMin   = "min"
	ReduceMax   = "max"
	ReduceSum   = "sum"
)

// ValidReduce is the set of supported range-query reducers.
var ValidReduce = map[string]bool{
	ReduceLast: true, ReduceFirst: true, ReduceAvg: true,
	ReduceMin: true, ReduceMax: true, ReduceSum: true,
}

// validate checks durations, query type, and the range-query block.
func (c KromgoConfig) validate() error {
	if s := c.Defaults.Timeseries.MaxDuration; s != "" {
		if _, err := ParseDuration(s); err != nil {
			return fmt.Errorf("defaults.timeseries.maxDuration: %w", err)
		}
	}
	for _, m := range c.Metrics {
		if m.Timeseries != nil && m.Timeseries.MaxDuration != "" {
			if _, err := ParseDuration(m.Timeseries.MaxDuration); err != nil {
				return fmt.Errorf("metric %q timeseries.maxDuration: %w", m.Name, err)
			}
		}
		if err := m.validate(); err != nil {
			return err
		}
	}
	return nil
}

// validate checks a metric's query type and range-query block.
func (m Metric) validate() error {
	switch m.Type {
	case "", TypeInstant:
		if m.Range != nil {
			return fmt.Errorf("metric %q: range block is only valid with type: range", m.Name)
		}
		return nil
	case TypeRange:
		if m.Range == nil || m.Range.Last == "" {
			return fmt.Errorf("metric %q: type range requires range.last", m.Name)
		}
		for name, val := range map[string]string{"last": m.Range.Last, "offset": m.Range.Offset, "step": m.Range.Step} {
			if val == "" {
				continue
			}
			if _, err := ParseDuration(val); err != nil {
				return fmt.Errorf("metric %q range.%s: %w", m.Name, name, err)
			}
		}
		if m.Range.Reduce != "" && !ValidReduce[m.Range.Reduce] {
			return fmt.Errorf("metric %q range.reduce: unknown reducer %q", m.Name, m.Range.Reduce)
		}
		return nil
	default:
		return fmt.Errorf("metric %q: unknown type %q (want %q or %q)", m.Name, m.Type, TypeInstant, TypeRange)
	}
}
