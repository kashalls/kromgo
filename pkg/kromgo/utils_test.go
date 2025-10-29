package kromgo

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

func TestGetColorConfig_MatchingRange(t *testing.T) {
	colors := []configuration.MetricColor{
		{Min: 0, Max: 10, Color: "blue", ValueOverride: "low"},
		{Min: 11, Max: 20, Color: "green", ValueOverride: "medium"},
		{Min: 21, Max: 30, Color: "red", ValueOverride: "high"},
	}

	value := 15.0

	result := GetColorConfig(colors, value)

	expected := configuration.MetricColor{Min: 11, Max: 20, Color: "green", ValueOverride: "medium"}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_ExactMatch(t *testing.T) {
	colors := []configuration.MetricColor{
		{Min: 0, Max: 10, Color: "blue"},
		{Min: 10, Max: 20, Color: "green"},
	}

	value := 10.0

	result := GetColorConfig(colors, value)

	expected := configuration.MetricColor{Min: 10, Max: 20, Color: "green"}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_NoMatch(t *testing.T) {
	colors := []configuration.MetricColor{
		{Min: 0, Max: 10, Color: "blue"},
		{Min: 11, Max: 20, Color: "green"},
	}

	value := 25.0

	result := GetColorConfig(colors, value)

	expected := configuration.MetricColor{Min: 25, Max: 25}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_EmptyColors(t *testing.T) {
	colors := []configuration.MetricColor{}

	value := 10.0

	result := GetColorConfig(colors, value)

	expected := configuration.MetricColor{Min: 10, Max: 10}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_ValueBelowMin(t *testing.T) {
	colors := []configuration.MetricColor{
		{Min: 10, Max: 20, Color: "green"},
		{Min: 21, Max: 30, Color: "red"},
	}

	value := 5.0

	result := GetColorConfig(colors, value)

	expected := configuration.MetricColor{Min: 5, Max: 5}
	assert.Equal(t, expected, result)
}

func TestGetColorConfig_ValueAboveMax(t *testing.T) {
	colors := []configuration.MetricColor{
		{Min: 0, Max: 10, Color: "blue"},
		{Min: 11, Max: 20, Color: "green"},
	}

	value := 25.0

	result := GetColorConfig(colors, value)

	expected := configuration.MetricColor{Min: 25, Max: 25}
	assert.Equal(t, expected, result)
}

func TestExtractLabelValue_LabelExists(t *testing.T) {
	vector := model.Vector{
		&model.Sample{
			Metric: model.Metric{
				"label1": "value1",
				"label2": "value2",
			},
		},
	}

	labelName := "label1"
	expectedValue := "value1"

	value, err := ExtractLabelValue(vector, labelName)

	assert.NoError(t, err)
	assert.Equal(t, expectedValue, value)
}

func TestExtractLabelValue_LabelDoesNotExist(t *testing.T) {
	vector := model.Vector{
		&model.Sample{
			Metric: model.Metric{
				"label1": "value1",
			},
		},
	}

	labelName := "label2"

	value, err := ExtractLabelValue(vector, labelName)

	assert.Error(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, "label 'label2' not found in the query result", err.Error())
}

func TestExtractLabelValue_EmptyVector(t *testing.T) {
	vector := model.Vector{}
	labelName := "label1"

	value, err := ExtractLabelValue(vector, labelName)

	assert.Error(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, "label 'label1' not found in the query result", err.Error())
}

func TestExtractLabelValue_LabelEmptyValue(t *testing.T) {
	vector := model.Vector{
		&model.Sample{
			Metric: model.Metric{
				"label1": "", // Empty string value for the label
			},
		},
	}

	labelName := "label1"
	expectedValue := ""

	value, err := ExtractLabelValue(vector, labelName)

	assert.NoError(t, err)
	assert.Equal(t, expectedValue, value)
}

func TestExtractRequestParams_URLParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics/test?format=json&style=flat", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("metric", "test")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	metric, format, style := ExtractRequestParams(req)

	assert.Equal(t, "test", metric)
	assert.Equal(t, "json", format)
	assert.Equal(t, "flat", style)
}

func TestExtractRequestParams_QueryParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics/query?metric=actual_metric&format=badge&style=plastic", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("metric", "query")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	metric, format, style := ExtractRequestParams(req)

	assert.Equal(t, "actual_metric", metric)
	assert.Equal(t, "badge", format)
	assert.Equal(t, "plastic", style)
}

func TestExtractRequestParams_NoParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics/test", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("metric", "test")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	metric, format, style := ExtractRequestParams(req)

	assert.Equal(t, "test", metric)
	assert.Equal(t, "", format)
	assert.Equal(t, "", style)
}

func TestExtractRequestParams_EmptyMetric(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics/?format=json", nil)
	rctx := chi.NewRouteContext()
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	metric, format, style := ExtractRequestParams(req)

	assert.Equal(t, "", metric)
	assert.Equal(t, "json", format)
	assert.Equal(t, "", style)
}
