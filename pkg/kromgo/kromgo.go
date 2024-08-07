package kromgo

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/log"
	"github.com/kashalls/kromgo/cmd/kromgo/init/prometheus"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

func KromgoRequestHandler(w http.ResponseWriter, r *http.Request, config configuration.Config) {
	requestMetric := chi.URLParam(r, "metric")
	requestFormat := r.URL.Query().Get("format")

	metric, exists := configuration.ProcessedMetrics[requestMetric]

	if !exists {
		requestLog(r).Error("metric not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Run the Prometheus query
	promResult, warnings, err := prometheus.Papi.Query(r.Context(), metric.Query, time.Now())
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error executing metric query")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(warnings) > 0 {
		for _, warning := range warnings {
			requestLog(r).With(zap.String("warning", warning)).Warn("encountered warnings while executing metric query")
		}
	}
	jsonResult, err := json.Marshal(promResult)
	requestLog(r).With(zap.String("result", string(jsonResult))).Debug("query result")
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("could not convert query result to json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(jsonResult) <= 0 {
		requestLog(r).Error("query returned no results")
	}

	if requestFormat == "raw" {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResult)
		return
	}

	prometheusData := promResult.(model.Vector)
	resultValue := float64(prometheusData[0].Value)
	colorConfig := GetColorConfig(metric.Colors, resultValue)

	var customResponse string = strconv.FormatFloat(resultValue, 'f', -1, 64)
	if len(metric.Label) > 0 {
		labelValue, err := ExtractLabelValue(prometheusData, metric.Label)
		if err != nil {
			requestLog(r).With(zap.String("label", metric.Label), zap.Error(err)).Error("label was not found in query result")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		customResponse = labelValue
	}
	if len(colorConfig.ValueOverride) > 0 {
		customResponse = colorConfig.ValueOverride
	}

	data := map[string]interface{}{
		"schemaVersion": 1,
		"label":  "",
		"message": metric.Prefix + customResponse + metric.Suffix,
	}

	if colorConfig.Color != "" {
		data["color"] = colorConfig.Color
	}

	jsonResponse, err := json.Marshal(data)
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error converting data to json response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}


func DeprecatedRequestHandler(w http.ResponseWriter, r *http.Request, config configuration.Config) {
	// Get the metric name from the query parameter
	metricName := r.URL.Query().Get("metric")
	responseFormat := r.URL.Query().Get("format")

	// Find the corresponding metric configuration
	var metric configuration.Metric
	for _, configMetric := range config.Metrics {
		if configMetric.Name == metricName {
			metric = configMetric
			break
		}
	}

	// If metric not found, return an error
	if metric.Query == "" {
		slog.Error(
			"metric not found",
			slog.String("ip", r.RemoteAddr),
			slog.String("metric", metric.Name),
		)
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	// Run the Prometheus query
	result, warnings, err := prometheus.Papi.Query(r.Context(), metric.Query, time.Now())
	if err != nil {
		slog.Error(
			"error executing query",
			slog.String("ip", r.RemoteAddr),
			slog.String("metric", metric.Name),
			"error", err,
		)
		http.Error(w, fmt.Sprintf("Error executing query: %s", err), http.StatusInternalServerError)
		return
	}

	if len(warnings) > 0 {
		fmt.Println("Warnings while executing query:", warnings)
	}

	// Convert the result to JSON
	jsonResult, err := json.Marshal(result)
	slog.Debug(
		"query result",
		slog.String("ip", r.RemoteAddr),
		slog.String("metric", metric.Name),
		slog.String("query", metric.Query),
		slog.String("result", string(jsonResult)),
	)
	if err != nil {
		slog.Error(
			"could not convert to json",
			slog.String("ip", r.RemoteAddr),
			slog.String("metric", metric.Name),
			"error", err,
		)
		http.Error(w, fmt.Sprintf("Error converting result to JSON: %s", err), http.StatusInternalServerError)
		return
	}

	if len(jsonResult) <= 0 {
		slog.Error(
			"query returned no results",
			slog.String("ip", r.RemoteAddr),
			slog.String("metric", metric.Name),
			slog.String("query", metric.Query),
		)
		http.Error(w, "Query returned no results", http.StatusNotFound)
		return
	}

	if responseFormat == "raw" {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResult)
		return
	} else {

		responseResult := result.(model.Vector)
		resultValue := float64(responseResult[0].Value)
		colorConfig := GetColorConfig(metric.Colors, resultValue)

		var whatAmIShowing string = strconv.FormatFloat(resultValue, 'f', -1, 64)

		if len(metric.Label) > 0 {
			value, err := ExtractLabelValue(responseResult, metric.Label)
			if err != nil {
				http.Error(w, "Label was not present in query.", http.StatusBadGateway)
				slog.Error(
					"label was not found in query result",
					slog.String("ip", r.RemoteAddr),
					slog.String("metric", metric.Name),
					"label", metric.Label,
				)
				return
			}
			whatAmIShowing = value
		}

		if len(colorConfig.ValueOverride) > 0 {
			whatAmIShowing = colorConfig.ValueOverride
		}

		message := metric.Prefix + whatAmIShowing + metric.Suffix

		data := map[string]interface{}{
			"schemaVersion": 1,
			"label":         metricName,
			"message":       message,
		}

		if colorConfig.Color != "" {
			data["color"] = colorConfig.Color
		}

		// Convert the data to JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "Error converting to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

func requestLog(r *http.Request) *zap.Logger {
	requestMetric := chi.URLParam(r, "metric")
	requestFormat := r.URL.Query().Get("format")

	return log.With(zap.String("req_method", r.Method), zap.String("req_path", r.URL.Path), zap.String("metric", requestMetric), zap.String("format", requestFormat))
}
