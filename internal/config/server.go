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

	// ServerReadTimeout / ServerWriteTimeout bound reading a request and writing its
	// response on the public listener; the defaults harden against slow-client
	// connection holding. WriteTimeout must exceed QueryTimeout so a slow upstream
	// isn't cut off mid-render. Set either to "0" to disable (no deadline).
	ServerReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"15s"`
	ServerWriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"60s"`
	ServerLogging      bool          `env:"SERVER_LOGGING"`

	// QueryTimeout bounds each outbound Prometheus query.
	QueryTimeout time.Duration `env:"QUERY_TIMEOUT" envDefault:"30s"`
}

// LoadServer reads ServerConfig from the environment.
func LoadServer() (ServerConfig, error) {
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}
