package prometheus

import (
	"fmt"
	"os"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var Papi v1.API

func Init(config configuration.KromgoConfig) (v1.API, error) {
	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL != "" {
		config.Prometheus = prometheusURL
	}

	if len(config.Prometheus) == 0 {
		return nil, fmt.Errorf("no url pointing to a prometheus instance was provided")
	}

	client, err := api.NewClient(api.Config{
		Address: config.Prometheus,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus client: %s", err)
	}

	Papi = v1.NewAPI(client)
	return Papi, nil
}
