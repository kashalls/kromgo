package configuration

import (
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/kashalls/kromgo/cmd/kromgo/init/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	ServerHost string `env:"SERVER_HOST" envDefault:"localhost"`
	ServerPort int    `env:"SERVER_PORT" envDefault:"8080"`

	HealthHost string `env:"HEALTH_HOST" envDefault:"localhost"`
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
}

type Metric struct {
	Name   string        `yaml:"name" json:"name"`
	Query  string        `yaml:"query" json:"query"`
	Label  string        `yaml:"label,omitempty" json:"label,omitempty"`
	Prefix string        `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Suffix string        `yaml:"suffix,omitempty" json:"suffix,omitempty"`
	Colors []MetricColor `yaml:"colors,omitempty" json:"colors,omitempty"`
}

type MetricColor struct {
	Min           float64 `yaml:"min" json:"min"`
	Max           float64 `yaml:"max" json:"max"`
	Color         string  `yaml:"color,omitempty" json:"color,omitempty"`
	ValueOverride string  `yaml:"valueOverride,omitempty" json:"valueOverride,omitempty"`
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

	ProcessedMetrics = preprocess(config.Metrics)
	return config
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
