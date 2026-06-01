package kromgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanizeBytes(t *testing.T) {
	// IEC binary units via go-humanize (spaced).
	assert.Equal(t, "1.0 KiB", humanizeBytes(1024))
	assert.Equal(t, "1.5 MiB", humanizeBytes(1572864))
	assert.Equal(t, "2.0 GiB", humanizeBytes(2147483648))
	assert.Equal(t, "512 B", humanizeBytes(512))
}

func TestHumanizeSIBytes(t *testing.T) {
	assert.Equal(t, "1.0 kB", humanizeSIBytes(1000))
	assert.Equal(t, "1.5 MB", humanizeSIBytes(1500000))
}

func TestHumanizeNumber(t *testing.T) {
	assert.Equal(t, "157,121", humanizeNumber(157121))
	assert.Equal(t, "1,000,000", humanizeNumber(1000000))
	assert.Equal(t, "1,234.56", humanizeNumber(1234.56))
	assert.Equal(t, "-1,234", humanizeNumber(-1234))
}

func TestHumanizeFloat(t *testing.T) {
	assert.Equal(t, "200", humanizeFloat(200))
	assert.Equal(t, "2.5", humanizeFloat(2.50))
	assert.Equal(t, "2", humanizeFloat(2.0))
	assert.Equal(t, "1234.567", humanizeFloat(1234.567))
}

func TestHumanizeDuration(t *testing.T) {
	const day = 86400.0
	// Sub-day: fine-grained, top-3 significant units.
	assert.Equal(t, "45s", humanizeDuration(45))
	assert.Equal(t, "1m30s", humanizeDuration(90))
	assert.Equal(t, "2h30m", humanizeDuration(9000)) // trailing 0s dropped
	assert.Equal(t, "1d2h3m", humanizeDuration(93780))
	// Long spans roll up to months ("mo") and years; sub-day noise drops off.
	assert.Equal(t, "1y3mo12d", humanizeDuration(467*day))   // 365 + 3*30 + 12
	assert.Equal(t, "5y8mo12d", humanizeDuration(179452910)) // 5y8mo12d1m50s, trimmed to top 3
	assert.Equal(t, "1y", humanizeDuration(365*day))         // exact year
	assert.Equal(t, "1mo5d", humanizeDuration(35*day))       // 1 month 5 days
	assert.Equal(t, "5d4h", humanizeDuration(5*day+4*3600))  // skips zero minutes/seconds
	// Edges.
	assert.Equal(t, "0s", humanizeDuration(0))
	assert.Equal(t, "0s", humanizeDuration(-5)) // clamped
}
