package kromgo

import (
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
		// humanizeBytes: IEC binary units via go-humanize (spaced).
		{"bytes KiB", humanizeBytes, 1024, "1.0 KiB"},
		{"bytes MiB", humanizeBytes, 1572864, "1.5 MiB"},
		{"bytes GiB", humanizeBytes, 2147483648, "2.0 GiB"},
		{"bytes B", humanizeBytes, 512, "512 B"},
		// humanizeSIBytes: SI decimal units.
		{"sibytes kB", humanizeSIBytes, 1000, "1.0 kB"},
		{"sibytes MB", humanizeSIBytes, 1500000, "1.5 MB"},
		// humanizeNumber: thousands separators.
		{"number int", humanizeNumber, 157121, "157,121"},
		{"number million", humanizeNumber, 1000000, "1,000,000"},
		{"number float", humanizeNumber, 1234.56, "1,234.56"},
		{"number negative", humanizeNumber, -1234, "-1,234"},
		// humanizeFloat: minimal float formatting.
		{"float int-valued", humanizeFloat, 200, "200"},
		{"float one decimal", humanizeFloat, 2.50, "2.5"},
		{"float whole", humanizeFloat, 2.0, "2"},
		{"float many decimals", humanizeFloat, 1234.567, "1234.567"},
		// humanizeDays: whole days, truncated and clamped at zero.
		{"days 69", humanizeDays, 69 * day, "69d"},
		{"days 544", humanizeDays, 544 * day, "544d"},
		{"days 6769", humanizeDays, 6769 * day, "6769d"},
		{"days truncates", humanizeDays, 1.5 * day, "1d"},
		{"days under a day", humanizeDays, 3600, "0d"},
		{"days clamped", humanizeDays, -5, "0d"},
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
