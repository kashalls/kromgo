package configuration

import "testing"

func TestValidateQueryTypes(t *testing.T) {
	cases := []struct {
		name    string
		metric  Metric
		wantErr bool
	}{
		{
			name:   "instant default",
			metric: Metric{Name: "a", Query: "up"},
		},
		{
			name:   "explicit instant",
			metric: Metric{Name: "a", Query: "up", QueryType: QueryTypeInstant},
		},
		{
			name:    "instant with range block",
			metric:  Metric{Name: "a", Query: "up", Range: &RangeConfig{Last: "7d", Step: "1h"}},
			wantErr: true,
		},
		{
			name:   "valid range",
			metric: Metric{Name: "a", Query: "up", QueryType: QueryTypeRange, Range: &RangeConfig{Last: "7d", Offset: "7d", Step: "1h", Reduce: "avg"}},
		},
		{
			name:    "range missing block",
			metric:  Metric{Name: "a", Query: "up", QueryType: QueryTypeRange},
			wantErr: true,
		},
		{
			name:    "range missing last",
			metric:  Metric{Name: "a", Query: "up", QueryType: QueryTypeRange, Range: &RangeConfig{Step: "1h"}},
			wantErr: true,
		},
		{
			name:    "range missing step",
			metric:  Metric{Name: "a", Query: "up", QueryType: QueryTypeRange, Range: &RangeConfig{Last: "7d"}},
			wantErr: true,
		},
		{
			name:    "range bad reduce",
			metric:  Metric{Name: "a", Query: "up", QueryType: QueryTypeRange, Range: &RangeConfig{Last: "7d", Step: "1h", Reduce: "median"}},
			wantErr: true,
		},
		{
			name:    "range bad offset",
			metric:  Metric{Name: "a", Query: "up", QueryType: QueryTypeRange, Range: &RangeConfig{Last: "7d", Step: "1h", Offset: "nope"}},
			wantErr: true,
		},
		{
			name:    "unknown type",
			metric:  Metric{Name: "a", Query: "up", QueryType: "series"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateQueryTypes(KromgoConfig{Metrics: []Metric{tc.metric}})
			if (err != nil) != tc.wantErr {
				t.Errorf("validateQueryTypes() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
