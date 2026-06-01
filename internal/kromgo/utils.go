package kromgo

import (
	"fmt"
	"slices"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
)

// GetColorConfig returns the last configured color range that contains value.
// When no range matches, it returns a zero-color config bounded to value.
func GetColorConfig(colors []config.MetricColor, value float64) config.MetricColor {
	for _, c := range slices.Backward(colors) {
		if value >= c.Min && value <= c.Max {
			return c
		}
	}
	return config.MetricColor{Min: value, Max: value}
}

// ExtractLabelValue returns the value of labelName from the first sample in the vector.
func ExtractLabelValue(vector model.Vector, labelName string) (string, error) {
	if len(vector) > 0 {
		if val, ok := vector[0].Metric[model.LabelName(labelName)]; ok {
			return string(val), nil
		}
	}
	return "", fmt.Errorf("label '%s' not found in the query result", labelName)
}
