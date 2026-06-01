package kromgo

import (
	"math"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/prometheus/common/model"
)

// reduceMatrix collapses each series in a range-query matrix to a single value via
// op, producing an instant vector the normal value pipeline can consume. Series with
// no finite samples are dropped (so an all-NaN series doesn't render as a value).
func reduceMatrix(matrix model.Matrix, op string) model.Vector {
	vector := make(model.Vector, 0, len(matrix))
	for _, stream := range matrix {
		if value, ts, ok := reduceSamples(stream.Values, op); ok {
			vector = append(vector, &model.Sample{Metric: stream.Metric, Value: value, Timestamp: ts})
		}
	}
	return vector
}

// reduceSamples reduces a series' finite samples to one value via op, returning
// ok=false when the series has no finite samples. The returned timestamp is the
// reduced sample's for first/last, otherwise the latest finite sample's.
func reduceSamples(values []model.SamplePair, op string) (model.SampleValue, model.Time, bool) {
	var (
		sum       float64
		count     int
		minV      = math.Inf(1)
		maxV      = math.Inf(-1)
		first     model.SamplePair
		last      model.SamplePair
		haveFirst bool
	)
	for _, s := range values {
		f := float64(s.Value)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		if !haveFirst {
			first, haveFirst = s, true
		}
		last = s
		sum += f
		minV = min(minV, f)
		maxV = max(maxV, f)
		count++
	}
	if count == 0 {
		return 0, 0, false
	}

	switch op {
	case config.ReduceFirst:
		return first.Value, first.Timestamp, true
	case config.ReduceMin:
		return model.SampleValue(minV), last.Timestamp, true
	case config.ReduceMax:
		return model.SampleValue(maxV), last.Timestamp, true
	case config.ReduceSum:
		return model.SampleValue(sum), last.Timestamp, true
	case config.ReduceAvg:
		return model.SampleValue(sum / float64(count)), last.Timestamp, true
	default: // ReduceLast (and the validated fallback)
		return last.Value, last.Timestamp, true
	}
}
