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
	Prometheus string            `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Metrics    []Metric          `yaml:"metrics" json:"metrics"`
	Badge      Badge             `yaml:"badge,omitempty" json:"badge,omitempty"`
	// Named Go template snippets that can be referenced by name in a metric's valueTemplate field.
	Templates  map[string]string `yaml:"templates,omitempty" json:"templates,omitempty"`
	// HideAll sets the default visibility for all metrics on the index page.
	// Defaults to true (all hidden) when not specified.
	HideAll    *bool             `yaml:"hideAll,omitempty" json:"hideAll,omitempty"`
	// History controls access to format=history requests.
	History    HistoryConfig     `yaml:"history,omitempty" json:"history,omitempty"`
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
