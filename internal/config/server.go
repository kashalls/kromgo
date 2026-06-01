package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// ServerConfig holds runtime server settings sourced from environment variables.
type ServerConfig struct {
	ServerHost string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	ServerPort int    `env:"SERVER_PORT" envDefault:"8080"`

	HealthHost string `env:"HEALTH_HOST" envDefault:"0.0.0.0"`
	HealthPort int    `env:"HEALTH_PORT" envDefault:"8888"`

	ServerReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT"`
	ServerWriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT"`
	ServerLogging      bool          `env:"SERVER_LOGGING"`

	// QueryTimeout bounds each outbound Prometheus query.
	QueryTimeout time.Duration `env:"QUERY_TIMEOUT" envDefault:"30s"`

	RatelimitEnable       bool          `env:"RATELIMIT_ENABLE"`
	RatelimitAll          bool          `env:"RATELIMIT_ALL"`
	RatelimitByRealIP     bool          `env:"RATELIMIT_BY_REAL_IP"`
	RatelimitRequestLimit int           `env:"RATELIMIT_REQUEST_LIMIT" envDefault:"100"`
	RatelimitWindowLength time.Duration `env:"RATELIMIT_WINDOW_LENGTH" envDefault:"1m"`
}

// LoadServer reads ServerConfig from the environment.
func LoadServer() (ServerConfig, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}
