package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricServed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kromgo",
			Name:      "metrics_served_total",
			Help:      "Total number of metrics served",
		},
		[]string{"metric", "format", "style", "status"},
	)

	MetricDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "kromgo",
			Name:      "metric_duration_seconds",
			Help:      "Duration taken to process metrics",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"metric", "format", "style"},
	)
)

func Init() {
	prometheus.MustRegister(MetricServed)
	prometheus.MustRegister(MetricDuration)
}
