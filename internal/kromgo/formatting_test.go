package kromgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanBytes(t *testing.T) {
	// IEC binary units via go-humanize (spaced).
	assert.Equal(t, "1.0 KiB", humanBytes(1024))
	assert.Equal(t, "1.5 MiB", humanBytes(1572864))
	assert.Equal(t, "2.0 GiB", humanBytes(2147483648))
	assert.Equal(t, "512 B", humanBytes(512))
}

func TestHumanSIBytes(t *testing.T) {
	assert.Equal(t, "1.0 kB", humanSIBytes(1000))
	assert.Equal(t, "1.5 MB", humanSIBytes(1500000))
}

func TestHumanizeThousands(t *testing.T) {
	assert.Equal(t, "157,121", humanizeThousands(157121))
	assert.Equal(t, "1,000,000", humanizeThousands(1000000))
	assert.Equal(t, "1,234.56", humanizeThousands(1234.56))
	assert.Equal(t, "-1,234", humanizeThousands(-1234))
}

func TestHumanizeFtoa(t *testing.T) {
	assert.Equal(t, "200", humanizeFtoa(200))
	assert.Equal(t, "2.5", humanizeFtoa(2.50))
	assert.Equal(t, "2", humanizeFtoa(2.0))
	assert.Equal(t, "1234.567", humanizeFtoa(1234.567))
}

func TestHumanDuration(t *testing.T) {
	assert.Equal(t, "45s", humanDuration(45))
	assert.Equal(t, "1m30s", humanDuration(90))
	assert.Equal(t, "2h30m", humanDuration(9000))
	assert.Equal(t, "1d2h3m", humanDuration(93780))
	assert.Equal(t, "0s", humanDuration(0))
}

func TestHumanizeAge(t *testing.T) {
	const day = 86400.0
	assert.Equal(t, "1y3m12d", humanizeAge(467*day)) // 365 + 3*30 + 12
	assert.Equal(t, "12d", humanizeAge(12*day))
	assert.Equal(t, "1y", humanizeAge(365*day))
	assert.Equal(t, "1m5d", humanizeAge(35*day))
	assert.Equal(t, "0d", humanizeAge(12*3600)) // under a day rounds down
	assert.Equal(t, "0d", humanizeAge(0))
	assert.Equal(t, "0d", humanizeAge(-5)) // clamped
}
