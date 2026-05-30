package kromgo

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
	"github.com/kashalls/kromgo/cmd/kromgo/init/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// executeMetricQuery fetches a metric's value from Prometheus according to its
// query type. Both instant and range queries normalize to a model.Vector so the
// downstream rendering pipeline (labels, colors, templates, badge/json) is shared.
func executeMetricQuery(ctx context.Context, metric configuration.Metric) (model.Vector, v1.Warnings, error) {
	switch metric.QueryType {
	case configuration.QueryTypeRange:
		return executeRangeQuery(ctx, metric)
	default:
		result, warnings, err := prometheus.Papi.Query(ctx, metric.Query, time.Now())
		if err != nil {
			return nil, warnings, err
		}
		vector, ok := result.(model.Vector)
		if !ok {
			return nil, warnings, fmt.Errorf("instant query did not return a vector")
		}
		return vector, warnings, nil
	}
}

// executeRangeQuery runs a range query over the configured window and reduces
// each series to a single sample, returning a synthetic vector.
func executeRangeQuery(ctx context.Context, metric configuration.Metric) (model.Vector, v1.Warnings, error) {
	start, end, step := rangeWindow(metric.Range, time.Now())

	result, warnings, err := prometheus.Papi.QueryRange(ctx, metric.Query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, warnings, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, warnings, fmt.Errorf("range query did not return a matrix")
	}

	reduce := metric.Range.Reduce
	if reduce == "" {
		reduce = "last"
	}

	vector := make(model.Vector, 0, len(matrix))
	for _, stream := range matrix {
		if len(stream.Values) == 0 {
			continue
		}
		vector = append(vector, &model.Sample{
			Metric:    stream.Metric,
			Value:     reduceSamples(stream.Values, reduce),
			Timestamp: stream.Values[len(stream.Values)-1].Timestamp,
		})
	}
	return vector, warnings, nil
}

// rangeWindow computes the query window from a RangeConfig relative to now.
// Durations are validated at config load, so parsing cannot fail here.
func rangeWindow(rc *configuration.RangeConfig, now time.Time) (start, end time.Time, step time.Duration) {
	last, _ := configuration.ParseDuration(rc.Last)
	step, _ = configuration.ParseDuration(rc.Step)

	var offset time.Duration
	if rc.Offset != "" {
		offset, _ = configuration.ParseDuration(rc.Offset)
	}

	end = now.Add(-offset)
	start = end.Add(-last)
	return start, end, step
}

// reduceSamples collapses a series of samples into a single value.
func reduceSamples(values []model.SamplePair, fn string) model.SampleValue {
	if len(values) == 0 {
		return 0
	}
	switch fn {
	case "first":
		return values[0].Value
	case "sum":
		var sum float64
		for _, v := range values {
			sum += float64(v.Value)
		}
		return model.SampleValue(sum)
	case "avg":
		var sum float64
		for _, v := range values {
			sum += float64(v.Value)
		}
		return model.SampleValue(sum / float64(len(values)))
	case "min":
		m := math.Inf(1)
		for _, v := range values {
			m = math.Min(m, float64(v.Value))
		}
		return model.SampleValue(m)
	case "max":
		m := math.Inf(-1)
		for _, v := range values {
			m = math.Max(m, float64(v.Value))
		}
		return model.SampleValue(m)
	default: // "last"
		return values[len(values)-1].Value
	}
}
