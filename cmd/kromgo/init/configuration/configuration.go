package configuration

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/kashalls/kromgo/cmd/kromgo/init/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// durationUnitRe matches a numeric value followed by a custom unit (y or d).
var durationUnitRe = regexp.MustCompile(`(\d+(?:\.\d+)?)(y|d)`)

// ParseDuration extends time.ParseDuration with support for days (d) and years (y).
// Units can be combined in any order: "1y30d", "7d12h", "3d5y", "2d".
// A value of "0" means unlimited (only meaningful for maxDuration config).
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	multipliers := map[string]time.Duration{
		"y": 365 * 24 * time.Hour,
		"d": 24 * time.Hour,
	}

	var total time.Duration
	remaining := s
	for _, m := range durationUnitRe.FindAllStringSubmatch(s, -1) {
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		total += time.Duration(float64(multipliers[m[2]]) * n)
		remaining = strings.Replace(remaining, m[0], "", 1)
	}

	if remaining != "" {
		d, err := time.ParseDuration(remaining)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		total += d
	}

	return total, nil
}

type ServerConfig struct {
	ServerHost string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	ServerPort int    `env:"SERVER_PORT" envDefault:"8080"`

	HealthHost string `env:"HEALTH_HOST" envDefault:"0.0.0.0"`
	HealthPort int    `env:"HEALTH_PORT" envDefault:"8888"`

	ServerReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT"`
	ServerWriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT"`
	ServerLogging      bool          `env:"SERVER_LOGGING"`

	RatelimitEnable       bool          `env:"RATELIMIT_ENABLE"`
	RatelimitAll          bool          `env:"RATELIMIT_ALL"`
	RatelimitByRealIP     bool          `env:"RATELIMIT_BY_REAL_IP"`
	RatelimitRequestLimit int           `env:"RATELIMIT_REQUEST_LIMIT" envDefault:"100"`
	RatelimitWindowLength time.Duration `env:"RATELIMIT_WINDOW_LENGTH" envDefault:"1m"`
}

// KromgoConfig struct for configuration environmental variables
type KromgoConfig struct {
	Prometheus string   `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Metrics    []Metric `yaml:"metrics" json:"metrics"`
	Badge      Badge    `yaml:"badge,omitempty" json:"badge,omitempty"`
	// Named Go template snippets that can be referenced by name in a metric's valueTemplate field.
	Templates map[string]string `yaml:"templates,omitempty" json:"templates,omitempty"`
	// HideAll sets the default visibility for all metrics on the index page.
	// Defaults to true (all hidden) when not specified.
	HideAll *bool `yaml:"hideAll,omitempty" json:"hideAll,omitempty"`
	// History controls access to format=history requests.
	History HistoryConfig `yaml:"history,omitempty" json:"history,omitempty"`
}

type HistoryConfig struct {
	// Enabled must be true to allow format=history requests. Defaults to false.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// MaxDuration caps the time window per request (e.g. "24h", "168h"). Defaults to "1h".
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty"`
}

type Metric struct {
	// The name of the metric. This is used in the HTTP Call
	Name string `yaml:"name" json:"name"`
	// The title of the metric to display. (Optional)
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// The prometheus query to run.
	Query string `yaml:"query" json:"query"`
	// QueryType selects how the metric value is fetched: "instant" (default) or "range".
	// A "range" query is evaluated over a window and reduced to a single value (see Range).
	QueryType string `yaml:"type,omitempty" json:"type,omitempty"`
	// Range configures the window, step, and reduction used when QueryType == "range".
	Range *RangeConfig `yaml:"range,omitempty" json:"range,omitempty"`
	// Fetch the value from this label in the prometheus query.
	Label string `yaml:"label,omitempty" json:"label,omitempty"`
	// Prefix the result of the query with this.
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	// Suffix the result of the query with this.
	Suffix string `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	// A Go template string applied to the result value before prefix/suffix are added.
	// Available functions: simplifyDays, humanBytes, humanDuration, toUpper, toLower, trim.
	// Example: "{{ . | simplifyDays }}" converts 1159 → 3y64d.
	ValueTemplate string `yaml:"valueTemplate,omitempty" json:"valueTemplate,omitempty"`
	// Add color.
	Colors []MetricColor `yaml:"colors,omitempty" json:"colors,omitempty"`
	// Hidden controls whether this metric appears on the index page.
	// If nil, the global HideAll setting is used (default: true).
	Hidden *bool `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	// History controls format=history access for this metric.
	// If nil, the global history settings are used.
	History *MetricHistoryConfig `yaml:"history,omitempty" json:"history,omitempty"`
}

// Query types accepted in a Metric's QueryType field.
const (
	QueryTypeInstant = "instant"
	QueryTypeRange   = "range"
)

// RangeConfig configures a range query that is reduced to a single value.
// The evaluated window is: end = now - Offset, start = end - Last.
type RangeConfig struct {
	// Last is the length of the window, e.g. "7d". Required for range queries.
	Last string `yaml:"last,omitempty" json:"last,omitempty"`
	// Offset shifts the whole window back in time, e.g. "7d". Defaults to "0" (window ends now).
	// Last "7d" with Offset "7d" covers 14d ago .. 7d ago.
	Offset string `yaml:"offset,omitempty" json:"offset,omitempty"`
	// Step is the resolution of the range query, e.g. "1h". Required for range queries.
	Step string `yaml:"step,omitempty" json:"step,omitempty"`
	// Reduce collapses each series to a single value: last|first|avg|min|max|sum. Defaults to "last".
	Reduce string `yaml:"reduce,omitempty" json:"reduce,omitempty"`
}

type MetricHistoryConfig struct {
	// Enabled overrides the global history.enabled for this metric.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	// MaxDuration overrides the global history.maxDuration for this metric (e.g. "24h").
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty"`
}

type MetricColor struct {
	Min           float64 `yaml:"min" json:"min"`
	Max           float64 `yaml:"max" json:"max"`
	Color         string  `yaml:"color,omitempty" json:"color,omitempty"`
	ValueOverride string  `yaml:"valueOverride,omitempty" json:"valueOverride,omitempty"`
}

type Badge struct {
	Font string `yaml:"font" json:"font"`
	Size int    `yaml:"size" json:"size"`
}

var ConfigPath = "/kromgo/config.yaml" // Default config file path
var ProcessedMetrics map[string]Metric

// Init sets up configuration by reading set environmental variables
func Init(configPath string) KromgoConfig {

	if configPath == "" {
		configPath = ConfigPath
	}

	// Read file from path.
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("error reading config file", zap.Error(err))
		os.Exit(1)
	}

	var config KromgoConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Error("error unmarshalling config yaml", zap.Error(err))
		os.Exit(1)
	}

	if err := validateHistoryDurations(config); err != nil {
		log.Error("invalid history configuration", zap.Error(err))
		os.Exit(1)
	}

	if err := validateQueryTypes(config); err != nil {
		log.Error("invalid query configuration", zap.Error(err))
		os.Exit(1)
	}

	ProcessedMetrics = preprocess(config.Metrics)
	return config
}

func validateHistoryDurations(config KromgoConfig) error {
	if s := config.History.MaxDuration; s != "" {
		if _, err := ParseDuration(s); err != nil {
			return fmt.Errorf("global history.maxDuration: %w", err)
		}
	}
	for _, m := range config.Metrics {
		if m.History != nil && m.History.MaxDuration != "" {
			if _, err := ParseDuration(m.History.MaxDuration); err != nil {
				return fmt.Errorf("metric %q history.maxDuration: %w", m.Name, err)
			}
		}
	}
	return nil
}

// validReduceFuncs is the set of reductions allowed in RangeConfig.Reduce.
var validReduceFuncs = map[string]bool{
	"last": true, "first": true, "avg": true, "min": true, "max": true, "sum": true,
}

func validateQueryTypes(config KromgoConfig) error {
	for _, m := range config.Metrics {
		switch m.QueryType {
		case "", QueryTypeInstant:
			if m.Range != nil {
				return fmt.Errorf("metric %q: range is only valid when type is %q", m.Name, QueryTypeRange)
			}
		case QueryTypeRange:
			if m.Range == nil {
				return fmt.Errorf("metric %q: type %q requires a range block", m.Name, QueryTypeRange)
			}
			d, err := ParseDuration(m.Range.Last)
			if err != nil {
				return fmt.Errorf("metric %q range.last: %w", m.Name, err)
			}
			if d <= 0 {
				return fmt.Errorf("metric %q range.last must be a positive duration", m.Name)
			}
			step, err := ParseDuration(m.Range.Step)
			if err != nil {
				return fmt.Errorf("metric %q range.step: %w", m.Name, err)
			}
			if step <= 0 {
				return fmt.Errorf("metric %q range.step must be a positive duration", m.Name)
			}
			if m.Range.Offset != "" {
				off, err := ParseDuration(m.Range.Offset)
				if err != nil {
					return fmt.Errorf("metric %q range.offset: %w", m.Name, err)
				}
				if off < 0 {
					return fmt.Errorf("metric %q range.offset must not be negative", m.Name)
				}
			}
			if m.Range.Reduce != "" && !validReduceFuncs[m.Range.Reduce] {
				return fmt.Errorf("metric %q range.reduce %q is not one of last|first|avg|min|max|sum", m.Name, m.Range.Reduce)
			}
		default:
			return fmt.Errorf("metric %q: unknown type %q (expected %q or %q)", m.Name, m.QueryType, QueryTypeInstant, QueryTypeRange)
		}
	}
	return nil
}

func InitServer() ServerConfig {
	cfg := ServerConfig{}
	if err := env.Parse(&cfg); err != nil {
		log.Error("error reading configuration from environment", zap.Error(err))
	}
	return cfg
}

func preprocess(metrics []Metric) map[string]Metric {
	reverseMap := make(map[string]Metric)
	for _, obj := range metrics {
		reverseMap[obj.Name] = obj
	}
	return reverseMap
}
