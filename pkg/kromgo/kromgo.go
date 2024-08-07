package kromgo

import (
	"encoding/json"
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

func KromgoRequestHandler(w http.ResponseWriter, r *http.Request, config configuration.KromgoConfig) {
	requestMetric := chi.URLParam(r, "metric")
	if requestMetric == "query" {
		requestMetric = r.URL.Query().Get("metric")
	}
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
		"label":         metric.Name,
		"message":       metric.Prefix + customResponse + metric.Suffix,
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

func requestLog(r *http.Request) *zap.Logger {
	requestMetric := chi.URLParam(r, "metric")
	requestFormat := r.URL.Query().Get("format")

	return log.With(zap.String("req_method", r.Method), zap.String("req_path", r.URL.Path), zap.String("metric", requestMetric), zap.String("format", requestFormat))
}
