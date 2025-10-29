package kromgo

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/prometheus/common/model"
)

func GetColorConfig(colors []configuration.MetricColor, value float64) configuration.MetricColor {
	for _, colorConfig := range colors {
		if value >= colorConfig.Min && value <= colorConfig.Max {
			return colorConfig
		}
	}

	// MetricColors is enabled, but the value does not have a corresponding value to it.
	// We return a default value here only if the result value falls outside the range.
	return configuration.MetricColor{
		Min: value,
		Max: value,
	}
}

func ExtractLabelValue(vector model.Vector, labelName string) (string, error) {
	// Extract label value from the first sample of the result
	if len(vector) > 0 {
		// Check if the label exists in the first sample
		if val, ok := vector[0].Metric[model.LabelName(labelName)]; ok {
			return string(val), nil
		}
	}

	// If label not found, return an error
	return "", fmt.Errorf("label '%s' not found in the query result", labelName)
}

// ExtractRequestParams extracts the metric name, format, and style from the HTTP request.
// If the metric is "query", it retrieves the metric name from the query parameters.
func ExtractRequestParams(r *http.Request) (metric, format, style string) {
	metric = chi.URLParam(r, "metric")
	if metric == "query" {
		metric = r.URL.Query().Get("metric")
	}
	format = r.URL.Query().Get("format")
	style = r.URL.Query().Get("style")
	return
}
