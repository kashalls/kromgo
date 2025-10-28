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

	MetricsNotFound = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kromgo",
			Name:      "metrics_not_found_total",
			Help:      "Total number of metrics not found",
		},
		[]string{"metric"},
	)

	MetricErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kromgo",
			Name:      "metric_errors_total",
			Help:      "Total number of errors encountered while processing metrics",
		},
		[]string{"metric", "error"},
	)
)

func Init() {
	prometheus.MustRegister(MetricServed)
	prometheus.MustRegister(MetricDuration)
	prometheus.MustRegister(MetricsNotFound)
	prometheus.MustRegister(MetricErrors)
}

func IncMetricsServed(metric, format, style string) {
	MetricServed.WithLabelValues(metric, format, style).Inc()
}

func IncMetricsNotFound(metric string) {
	MetricsNotFound.WithLabelValues(metric).Inc()
}

var MetricErrorNotFound = "Not Found"
var MetricErrorProcessingError = "Processing Error"
var MetricErrorBadgeGenerationError = "Badge Generation Error"

func IncMetricErrors(metric, errorType string) {
	MetricErrors.WithLabelValues(metric, errorType).Inc()
}
