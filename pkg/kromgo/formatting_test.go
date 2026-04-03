package kromgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplifyDays_YearsAndDays(t *testing.T) {
	assert.Equal(t, "3y64d", simplifyDays("1159"))
}

func TestSimplifyDays_DaysOnly(t *testing.T) {
	assert.Equal(t, "45d", simplifyDays("45"))
}

func TestSimplifyDays_ExactYears(t *testing.T) {
	assert.Equal(t, "1y0d", simplifyDays("365"))
}

func TestSimplifyDays_ZeroDays(t *testing.T) {
	assert.Equal(t, "0d", simplifyDays("0"))
}

func TestSimplifyDays_Float64Input(t *testing.T) {
	assert.Equal(t, "3y64d", simplifyDays(float64(1159)))
}

func TestSimplifyDays_InvalidInput(t *testing.T) {
	assert.Equal(t, "notanumber", simplifyDays("notanumber"))
}

func TestHumanBytes_Bytes(t *testing.T) {
	assert.Equal(t, "512B", humanBytes("512"))
}

func TestHumanBytes_Kilobytes(t *testing.T) {
	assert.Equal(t, "1.0KB", humanBytes("1024"))
}

func TestHumanBytes_Megabytes(t *testing.T) {
	assert.Equal(t, "1.5MB", humanBytes("1572864"))
}

func TestHumanBytes_Gigabytes(t *testing.T) {
	assert.Equal(t, "2.0GB", humanBytes("2147483648"))
}

func TestHumanBytes_InvalidInput(t *testing.T) {
	assert.Equal(t, "notanumber", humanBytes("notanumber"))
}

func TestHumanDuration_SecondsOnly(t *testing.T) {
	assert.Equal(t, "45s", humanDuration("45"))
}

func TestHumanDuration_MinutesAndSeconds(t *testing.T) {
	assert.Equal(t, "1m30s", humanDuration("90"))
}

func TestHumanDuration_HoursAndMinutes(t *testing.T) {
	assert.Equal(t, "2h30m", humanDuration("9000"))
}

func TestHumanDuration_DaysHoursMinutes(t *testing.T) {
	assert.Equal(t, "1d2h3m", humanDuration("93780"))
}

func TestHumanDuration_Zero(t *testing.T) {
	assert.Equal(t, "0s", humanDuration("0"))
}

func TestHumanDuration_InvalidInput(t *testing.T) {
	assert.Equal(t, "notanumber", humanDuration("notanumber"))
}

func TestHumanizeThousands_Large(t *testing.T) {
	assert.Equal(t, "157,121", humanizeThousands("157121"))
}

func TestHumanizeThousands_Small(t *testing.T) {
	assert.Equal(t, "999", humanizeThousands("999"))
}

func TestHumanizeThousands_Millions(t *testing.T) {
	assert.Equal(t, "1,000,000", humanizeThousands("1000000"))
}

func TestHumanizeThousands_Float64Input(t *testing.T) {
	assert.Equal(t, "157,121", humanizeThousands(float64(157121)))
}

func TestHumanizeThousands_WithDecimal(t *testing.T) {
	assert.Equal(t, "1,234.56", humanizeThousands("1234.56"))
}

func TestHumanizeThousands_Negative(t *testing.T) {
	assert.Equal(t, "-1,234", humanizeThousands("-1234"))
}

func TestHumanizeThousands_InvalidInput(t *testing.T) {
	assert.Equal(t, "notanumber", humanizeThousands("notanumber"))
}

func TestApplyValueTemplate_HumanizeThousands(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | humanizeThousands }}", "157121")
	assert.NoError(t, err)
	assert.Equal(t, "157,121", result)
}

func TestApplyValueTemplate_SimplifyDays(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | simplifyDays }}", "1159")
	assert.NoError(t, err)
	assert.Equal(t, "3y64d", result)
}

func TestApplyValueTemplate_HumanBytes(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | humanBytes }}", "1572864")
	assert.NoError(t, err)
	assert.Equal(t, "1.5MB", result)
}

func TestApplyValueTemplate_HumanDuration(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | humanDuration }}", "9000")
	assert.NoError(t, err)
	assert.Equal(t, "2h30m", result)
}

func TestApplyValueTemplate_ToUpper(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | toUpper }}", "v1.31.0")
	assert.NoError(t, err)
	assert.Equal(t, "V1.31.0", result)
}

func TestApplyValueTemplate_ToLower(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | toLower }}", "HEALTHY")
	assert.NoError(t, err)
	assert.Equal(t, "healthy", result)
}

func TestApplyValueTemplate_Trim(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . | trim }}", "  hello  ")
	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestApplyValueTemplate_InvalidTemplate_ReturnsOriginal(t *testing.T) {
	result, err := ApplyValueTemplate("{{ .invalid syntax }", "1159")
	assert.Error(t, err)
	assert.Equal(t, "1159", result)
}

func TestApplyValueTemplate_EmptyTemplate(t *testing.T) {
	result, err := ApplyValueTemplate("", "1159")
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestApplyValueTemplate_PassThrough(t *testing.T) {
	result, err := ApplyValueTemplate("{{ . }}", "unchanged")
	assert.NoError(t, err)
	assert.Equal(t, "unchanged", result)
}
