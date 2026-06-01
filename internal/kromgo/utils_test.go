package kromgo

import (
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func TestGetColorConfig_MatchingRange(t *testing.T) {
	colors := []config.MetricColor{
		{Min: 0, Max: 10, Color: "blue", Display: "low"},
		{Min: 11, Max: 20, Color: "green", Display: "medium"},
		{Min: 21, Max: 30, Color: "red", Display: "high"},
	}

	result := GetColorConfig(colors, 15.0)

	expected := config.MetricColor{Min: 11, Max: 20, Color: "green", Display: "medium"}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_ExactMatch(t *testing.T) {
	colors := []config.MetricColor{
		{Min: 0, Max: 10, Color: "blue"},
		{Min: 10, Max: 20, Color: "green"},
	}

	result := GetColorConfig(colors, 10.0)

	expected := config.MetricColor{Min: 10, Max: 20, Color: "green"}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_NoMatch(t *testing.T) {
	colors := []config.MetricColor{
		{Min: 0, Max: 10, Color: "blue"},
		{Min: 11, Max: 20, Color: "green"},
	}

	result := GetColorConfig(colors, 25.0)

	expected := config.MetricColor{Min: 25, Max: 25}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_EmptyColors(t *testing.T) {
	result := GetColorConfig([]config.MetricColor{}, 10.0)

	expected := config.MetricColor{Min: 10, Max: 10}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_ValueBelowMin(t *testing.T) {
	colors := []config.MetricColor{
		{Min: 10, Max: 20, Color: "green"},
		{Min: 21, Max: 30, Color: "red"},
	}

	result := GetColorConfig(colors, 5.0)

	expected := config.MetricColor{Min: 5, Max: 5}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_ValueAboveMax(t *testing.T) {
	colors := []config.MetricColor{
		{Min: 0, Max: 10, Color: "blue"},
		{Min: 11, Max: 20, Color: "green"},
	}

	result := GetColorConfig(colors, 25.0)

	expected := config.MetricColor{Min: 25, Max: 25}
	assert.Equal(t, expected, result)
}

func TestExtractLabelValue_LabelExists(t *testing.T) {
	vector := model.Vector{
		&model.Sample{Metric: model.Metric{"label1": "value1", "label2": "value2"}},
	}

	value, err := ExtractLabelValue(vector, "label1")

	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestExtractLabelValue_LabelDoesNotExist(t *testing.T) {
	vector := model.Vector{
		&model.Sample{Metric: model.Metric{"label1": "value1"}},
	}

	value, err := ExtractLabelValue(vector, "label2")

	assert.Error(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, "label 'label2' not found in the query result", err.Error())
}

func TestExtractLabelValue_EmptyVector(t *testing.T) {
	value, err := ExtractLabelValue(model.Vector{}, "label1")

	assert.Error(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, "label 'label1' not found in the query result", err.Error())
}

func TestExtractLabelValue_LabelEmptyValue(t *testing.T) {
	vector := model.Vector{
		&model.Sample{Metric: model.Metric{"label1": ""}},
	}

	value, err := ExtractLabelValue(vector, "label1")

	assert.NoError(t, err)
	assert.Equal(t, "", value)
}
