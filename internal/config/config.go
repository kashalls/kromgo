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
	// Range controls access to format=history and format=chart (range query) requests.
	Range RangeConfig `yaml:"range,omitempty" json:"range,omitempty"`
}

// RangeConfig holds the default settings for range-query (history/chart) requests.
type RangeConfig struct {
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
	// Range overrides defaults.range for this metric. If nil, the defaults apply.
	Range *MetricRangeConfig `yaml:"range,omitempty" json:"range,omitempty"`
	// CacheSeconds overrides defaults.cacheSeconds for this metric. If nil, the default applies.
	CacheSeconds *int `yaml:"cacheSeconds,omitempty" json:"cacheSeconds,omitempty"`
}

// MetricRangeConfig overrides the default RangeConfig for a single metric.
type MetricRangeConfig struct {
	// Enabled overrides defaults.range.enabled for this metric.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	// MaxDuration overrides defaults.range.maxDuration for this metric (e.g. "24h").
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

	var cfg KromgoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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

// validate checks that all configured durations parse.
func (c KromgoConfig) validate() error {
	if s := c.Defaults.Range.MaxDuration; s != "" {
		if _, err := ParseDuration(s); err != nil {
			return fmt.Errorf("defaults.range.maxDuration: %w", err)
		}
	}
	for _, m := range c.Metrics {
		if m.Range != nil && m.Range.MaxDuration != "" {
			if _, err := ParseDuration(m.Range.MaxDuration); err != nil {
				return fmt.Errorf("metric %q range.maxDuration: %w", m.Name, err)
			}
		}
	}
	return nil
}
