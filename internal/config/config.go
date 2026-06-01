// Package config defines kromgo's configuration model and loads it from YAML and
// the environment.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the config file location used when none is provided.
const DefaultPath = "/kromgo/config.yaml"

// KromgoConfig is the top-level YAML configuration.
type KromgoConfig struct {
	Prometheus string   `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Metrics    []Metric `yaml:"metrics" json:"metrics"`
	Badge      Badge    `yaml:"badge,omitempty" json:"badge,omitempty"`
	// Templates are named Go template snippets referenced by name in a metric's valueTemplate field.
	Templates map[string]string `yaml:"templates,omitempty" json:"templates,omitempty"`
	// HideAll sets the default visibility for all metrics on the index page.
	// Defaults to true (all hidden) when not specified.
	HideAll *bool `yaml:"hideAll,omitempty" json:"hideAll,omitempty"`
	// History controls access to format=history and format=chart requests.
	History HistoryConfig `yaml:"history,omitempty" json:"history,omitempty"`
}

// HistoryConfig holds the global settings for time-series (history/chart) requests.
type HistoryConfig struct {
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
	// ValueTemplate is a Go template applied to the value before prefix/suffix are added.
	// Available functions: simplifyDays, humanBytes, humanSIBytes, humanDuration,
	// humanizeThousands, toUpper, toLower, trim. Example: "{{ . | simplifyDays }}".
	ValueTemplate string `yaml:"valueTemplate,omitempty" json:"valueTemplate,omitempty"`
	// Colors assigns a response color based on the numeric value.
	Colors []MetricColor `yaml:"colors,omitempty" json:"colors,omitempty"`
	// Hidden controls whether this metric appears on the index page.
	// If nil, the global HideAll setting is used (default: true).
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// History overrides the global history settings for this metric. If nil, the global settings apply.
	History *MetricHistoryConfig `yaml:"history,omitempty" json:"history,omitempty"`
}

// MetricHistoryConfig overrides the global HistoryConfig for a single metric.
type MetricHistoryConfig struct {
	// Enabled overrides the global history.enabled for this metric.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	// MaxDuration overrides the global history.maxDuration for this metric (e.g. "24h").
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
	if s := c.History.MaxDuration; s != "" {
		if _, err := ParseDuration(s); err != nil {
			return fmt.Errorf("global history.maxDuration: %w", err)
		}
	}
	for _, m := range c.Metrics {
		if m.History != nil && m.History.MaxDuration != "" {
			if _, err := ParseDuration(m.History.MaxDuration); err != nil {
				return fmt.Errorf("metric %q history.maxDuration: %w", m.Name, err)
			}
		}
	}
	return nil
}
