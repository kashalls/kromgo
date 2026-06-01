package kromgo

import "github.com/prometheus/client_golang/prometheus"

var requestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "kromgo_requests_total",
		Help: "Total number of requests processed by kromgo, partitioned by metric name and format.",
	},
	[]string{"metric", "format"},
)

func init() {
	prometheus.MustRegister(requestsTotal)
}
