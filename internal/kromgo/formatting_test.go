package kromgo

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

const day = 86400.0

func TestHumanizers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		fn   func(float64) string
		in   float64
		want string
	}{
		// humanizeBytes: SI decimal units, no space, scaled values to one decimal.
		{"bytes B", humanizeBytes, 512, "512B"},
		{"bytes kB", humanizeBytes, 1000, "1kB"},
		{"bytes MB", humanizeBytes, 1500000, "1.5MB"},
		{"bytes MB rounds", humanizeBytes, 1572864, "1.6MB"},
		{"bytes GB", humanizeBytes, 2147483648, "2.1GB"},
		{"bytes whole MB", humanizeBytes, 1000000, "1MB"},
		{"bytes carry to next unit", humanizeBytes, 999999, "1MB"}, // 999.999kB rounds up → carry
		{"bytes carry boundary", humanizeBytes, 999950, "1MB"},
		{"bytes no false carry", humanizeBytes, 999949, "999.9kB"},
		// Non-finite inputs degrade to a clean string, not a unit-suffixed/comma-mangled one.
		{"bytes NaN", humanizeBytes, math.NaN(), "NaN"},
		{"bytes +Inf", humanizeBytes, math.Inf(1), "+Inf"},
		{"commas +Inf", humanizeCommas, math.Inf(1), "+Inf"},
		{"commas NaN", humanizeCommas, math.NaN(), "NaN"},
		{"float NaN", humanizeFloat, math.NaN(), "NaN"},
		// humanizeCommas: thousands separators.
		{"commas int", humanizeCommas, 157121, "157,121"},
		{"commas million", humanizeCommas, 1000000, "1,000,000"},
		{"commas float", humanizeCommas, 1234.56, "1,234.56"},
		{"commas negative", humanizeCommas, -1234, "-1,234"},
		{"commas small", humanizeCommas, 42, "42"},
		// humanizeFloat: minimal float formatting.
		{"float int-valued", humanizeFloat, 200, "200"},
		{"float one decimal", humanizeFloat, 2.50, "2.5"},
		{"float whole", humanizeFloat, 2.0, "2"},
		{"float many decimals", humanizeFloat, 1234.567, "1234.567"},
		// humanizeDurationDays: whole days, truncated and clamped at zero.
		{"days 69", humanizeDurationDays, 69 * day, "69d"},
		{"days 544", humanizeDurationDays, 544 * day, "544d"},
		{"days 6769", humanizeDurationDays, 6769 * day, "6769d"},
		{"days truncates", humanizeDurationDays, 1.5 * day, "1d"},
		{"days under a day", humanizeDurationDays, 3600, "0d"},
		{"days clamped", humanizeDurationDays, -5, "0d"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.fn(tc.in))
		})
	}
}

func TestHumanizeDuration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   float64
		want string
	}{
		// Sub-day: fine-grained, top-3 significant units.
		{"seconds", 45, "45s"},
		{"minutes", 90, "1m30s"},
		{"trailing zeros dropped", 9000, "2h30m"},
		{"day hour minute", 93780, "1d2h3m"},
		// Long spans roll up to months ("mo") and years; sub-day noise drops off.
		{"year months days", 467 * day, "1y3mo12d"},  // 365 + 3*30 + 12
		{"trimmed to top 3", 179452910, "5y8mo12d"},  // 5y8mo12d1m50s
		{"exact year", 365 * day, "1y"},              // exact year
		{"month and days", 35 * day, "1mo5d"},        // 1 month 5 days
		{"skips zero units", 5*day + 4*3600, "5d4h"}, // skips zero minutes/seconds
		// Edges.
		{"zero", 0, "0s"},
		{"negative clamped", -5, "0s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, humanizeDuration(tc.in))
		})
	}
}
