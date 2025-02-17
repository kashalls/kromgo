package kromgo

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/essentialkaos/go-badge"
	"github.com/go-chi/chi/v5"
	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/log"
	"github.com/kashalls/kromgo/cmd/kromgo/init/prometheus"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

type KromgoHandler struct {
	Config         configuration.KromgoConfig
	BadgeGenerator *badge.Generator
}

// NewKromgoHandler initializes the handler with the necessary dependencies
func NewKromgoHandler(config configuration.KromgoConfig) (*KromgoHandler, error) {
	var badgeGenerator *badge.Generator

	font := config.Badge.Font
	if font == "" {
		font = "Verdana.ttf"
	}

	size := config.Badge.Size
	if size <= 0 {
		size = 11
	}

	badgeGenerator, err := badge.NewGenerator(font, size)
	if err != nil {
		return nil, err
	}

	return &KromgoHandler{
		Config:         config,
		BadgeGenerator: badgeGenerator,
	}, nil
}

func (h *KromgoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestMetric := chi.URLParam(r, "metric")
	if requestMetric == "query" {
		requestMetric = r.URL.Query().Get("metric")
	}
	requestFormat := r.URL.Query().Get("format")
	badgeStyle := r.URL.Query().Get("style")

	if requestFormat == "badge" && h.BadgeGenerator == nil {
		HandleError(w, r, requestMetric, "Format badge is not configured", http.StatusInternalServerError)
		return
	}

	metric, exists := configuration.ProcessedMetrics[requestMetric]

	if !exists {
		requestLog(r).Error("metric not found")
		HandleError(w, r, requestMetric, "Not Found", http.StatusNotFound)
		return
	}

	// Run the Prometheus query
	// potentially utilize withlimit or withtimeout
	promResult, warnings, err := prometheus.Papi.Query(r.Context(), metric.Query, time.Now())
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error executing metric query")
		w.WriteHeader(http.StatusInternalServerError)
		HandleError(w, r, requestMetric, "Query Error", http.StatusInternalServerError)
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
		HandleError(w, r, requestMetric, "Query Error", http.StatusInternalServerError)
		return
	}

	if len(jsonResult) <= 0 {
		requestLog(r).Error("query returned no results")
		HandleError(w, r, requestMetric, "No Data", http.StatusOK)
		return
	}

	if requestFormat == "raw" {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResult)
		return
	}

	prometheusData := promResult.(model.Vector)
	log.Debug("prometheus returned data", zap.Any("data", prometheusData))

	var colorConfig configuration.MetricColor
	var response string 
	if len(prometheusData) > 0 {
		resultValue := float64(prometheusData[0].Value)
		colorConfig = GetColorConfig(metric.Colors, resultValue)
		response = strconv.FormatFloat(resultValue, 'f', -1, 64)
	} else {
		colorConfig = configuration.MetricColor{}
		response = "metric returned no data"
	}

	if len(metric.Label) > 0 {
		labelValue, err := ExtractLabelValue(prometheusData, metric.Label)
		if err != nil {
			requestLog(r).With(zap.String("label", metric.Label), zap.Error(err)).Error("label was not found in query result")
			HandleError(w, r, requestMetric, "No Data", http.StatusOK)
			return
		}
		response = labelValue
	}
	if len(colorConfig.ValueOverride) > 0 {
		response = colorConfig.ValueOverride
	}

	message := metric.Prefix + response + metric.Suffix

	title := metric.Name
	if metric.Title != "" {
		title = metric.Title
	}

	if requestFormat == "badge" {
		hex := colorNameToHex(colorConfig.Color)

		w.Header().Set("Content-Type", "image/svg+xml")

		if badgeStyle == "plastic" {
			w.Write(h.BadgeGenerator.GeneratePlastic(title, message, hex))
			return
		}
		if badgeStyle == "flat-square" {
			w.Write(h.BadgeGenerator.GenerateFlatSquare(title, message, hex))
			return
		}
		//if badgeStyle == "flat" || badgeStyle == "" {
		//	w.Write(h.BadgeGenerator.GenerateFlat(title, message, hex))
		//	return
		//}

		w.Write(h.BadgeGenerator.GenerateFlat(title, message, hex))
		return
	}

	data := map[string]interface{}{
		"schemaVersion": 1,
		"label":         title,
		"message":       message,
	}

	if colorConfig.Color != "" {
		data["color"] = colorConfig.Color
	}

	jsonResponse, err := json.Marshal(data)
	if err != nil {
		requestLog(r).With(zap.Error(err)).Error("error converting data to json response")
		HandleError(w, r, requestMetric, "Error", http.StatusInternalServerError)
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

func colorNameToHex(colorName string) string {
	if strings.HasPrefix(colorName, "#") {
		return colorName
	}

	switch colorName {
	case "":
		return badge.COLOR_BLUE
	case "blue":
		return badge.COLOR_BLUE
	case "brightgreen":
		return badge.COLOR_BRIGHTGREEN
	case "green":
		return badge.COLOR_GREEN
	case "grey":
		return badge.COLOR_GREY
	case "lightgrey":
		return badge.COLOR_LIGHTGREY
	case "orange":
		return badge.COLOR_ORANGE
	case "red":
		return badge.COLOR_RED
	case "yellow":
		return badge.COLOR_YELLOW
	case "yellowgreen":
		return badge.COLOR_YELLOWGREEN
	case "success":
		return badge.COLOR_SUCCESS
	case "important":
		return badge.COLOR_IMPORTANT
	case "critical":
		return badge.COLOR_CRITICAL
	case "informational":
		return badge.COLOR_INFORMATIONAL
	case "inactive":
		return badge.COLOR_INACTIVE
	default:
		return badge.COLOR_GREEN
	}
}
