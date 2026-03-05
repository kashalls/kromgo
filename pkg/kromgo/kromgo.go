package kromgo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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
	Config    configuration.KromgoConfig
	badgePool sync.Pool
}

// NewKromgoHandler initializes the handler with the necessary dependencies
func NewKromgoHandler(config configuration.KromgoConfig) (*KromgoHandler, error) {
	font := config.Badge.Font
	if font == "" {
		font = "Verdana.ttf"
	}

	size := config.Badge.Size
	if size <= 0 {
		size = 11
	}

	fontData, err := os.ReadFile(font)
	if err != nil {
		return nil, fmt.Errorf("failed to read font file: %w", err)
	}
	// Verify the font parses correctly at startup.
	if _, err := badge.NewGeneratorFromBytes(fontData, size); err != nil {
		return nil, err
	}

	return &KromgoHandler{
		Config: config,
		badgePool: sync.Pool{
			New: func() any {
				gen, _ := badge.NewGeneratorFromBytes(fontData, size)
				return gen
			},
		},
	}, nil
}

func (h *KromgoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestMetric := chi.URLParam(r, "metric")
	if requestMetric == "" {
		HandleError(w,r, requestMetric, "A valid metric name must be passed /{metric}", http.StatusBadRequest)
		return
	}
	if requestMetric == "query" {
		requestMetric = r.URL.Query().Get("metric")
	}
	requestFormat := r.URL.Query().Get("format")
	if requestFormat == "" {
		requestFormat = "json"
	}
	badgeStyle := r.URL.Query().Get("style")

	defer func() {
		requestsTotal.WithLabelValues(requestMetric, requestFormat).Inc()
	}()


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

	if metric.ValueTemplate != "" {
		tmplStr := metric.ValueTemplate
		if resolved, ok := h.Config.Templates[tmplStr]; ok {
			tmplStr = resolved
		}
		formatted, err := ApplyValueTemplate(tmplStr, response)
		if err != nil {
			requestLog(r).With(zap.Error(err)).Error("failed to apply value template")
		}
		response = formatted
	}

	message := metric.Prefix + response + metric.Suffix

	title := metric.Name
	if metric.Title != "" {
		title = metric.Title
	}

	if requestFormat == "badge" {
		gen := h.badgePool.Get().(*badge.Generator)
		defer h.badgePool.Put(gen)

		hex := colorNameToHex(colorConfig.Color)
		w.Header().Set("Content-Type", "image/svg+xml")
		switch badgeStyle {
		case "plastic":
			w.Write(gen.GeneratePlastic(title, message, hex))
		case "flat-square":
			w.Write(gen.GenerateFlatSquare(title, message, hex))
		default:
			w.Write(gen.GenerateFlat(title, message, hex))
		}
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
