package kromgo

import "github.com/prometheus/client_golang/prometheus"

var requestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "kromgo_requests_total",
		Help: "Total requests processed by kromgo, partitioned by endpoint kind, id, and format.",
	},
	[]string{"kind", "id", "format"},
)

func init() {
	prometheus.MustRegister(requestsTotal)
}
