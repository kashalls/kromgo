package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kashalls/kromgo/pkg/kromgo"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := statusRecorder{ResponseWriter: w, status: 200}

		next.ServeHTTP(&rec, r)

		duration := time.Since(start).Seconds()
		metric, format, style := kromgo.ExtractRequestParams(r)

		// We do not want to collect metrics for 404 responses. This prevents someone from creating a metric
		// that does not exist and skewing our metrics with a high number of 404s.
		if rec.status == http.StatusNotFound {
			return
		}

		if metric == "" {
			metric = "unknown"
		}

		MetricDuration.WithLabelValues(metric, format, style).Observe(duration)
		MetricServed.WithLabelValues(metric, format, style, strconv.Itoa(rec.status)).Inc()
	})
}
