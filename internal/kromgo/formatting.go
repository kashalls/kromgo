package kromgo

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/template"
)

var templateFuncs = template.FuncMap{
	"simplifyDays":      simplifyDays,
	"humanBytes":        humanBytes,
	"humanSIBytes":      humanSIBytes,
	"humanDuration":     humanDuration,
	"humanizeThousands": humanizeThousands,
	"toUpper":           strings.ToUpper,
	"toLower":           strings.ToLower,
	"trim":              strings.TrimSpace,
}

// ApplyValueTemplate executes the given Go template string with value as the dot (.) data.
// Returns the formatted string and nil on success, or the original value and an error if the
// template fails to parse or execute.
func ApplyValueTemplate(tmplStr string, value string) (string, error) {
	tmpl, err := template.New("value").Funcs(templateFuncs).Parse(tmplStr)
	if err != nil {
		return value, fmt.Errorf("failed to parse value template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, value); err != nil {
		return value, fmt.Errorf("failed to execute value template: %w", err)
	}
	return buf.String(), nil
}

// toFloat converts a string, int, or float64 to float64.
func toFloat(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(val), 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// simplifyDays converts a day count into a compact human-readable string.
// For example, 1159 days becomes "3y64d".
func simplifyDays(v any) string {
	f, err := toFloat(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	days := int(math.Round(f))
	years := days / 365
	remaining := days % 365
	if years > 0 {
		return fmt.Sprintf("%dy%dd", years, remaining)
	}
	return fmt.Sprintf("%dd", remaining)
}

// humanBytes converts a byte count into a human-readable size string using
// IEC binary units (powers of 1024). For example, 1572864 becomes "1.5MiB".
// For SI decimal units (powers of 1000, kB/MB/GB...) use humanSIBytes.
func humanBytes(v any) string {
	f, err := toFloat(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return scaleBytes(f, 1024, []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"})
}

// humanSIBytes converts a byte count into a human-readable size string using
// SI decimal units (powers of 1000). For example, 1500000 becomes "1.5MB".
// For IEC binary units (powers of 1024, KiB/MiB/GiB...) use humanBytes.
func humanSIBytes(v any) string {
	f, err := toFloat(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return scaleBytes(f, 1000, []string{"B", "kB", "MB", "GB", "TB", "PB"})
}

func scaleBytes(f float64, base float64, units []string) string {
	i := 0
	for f >= base && i < len(units)-1 {
		f /= base
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d%s", int(math.Round(f)), units[0])
	}
	return fmt.Sprintf("%.1f%s", f, units[i])
}

// humanizeThousands formats a number with comma thousands separators.
// For example, 157121 becomes "157,121".
func humanizeThousands(v any) string {
	f, err := toFloat(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	// Format as integer if no fractional part, otherwise keep decimals.
	var intPart int64
	var fracStr string
	if f == math.Trunc(f) {
		intPart = int64(math.Round(f))
	} else {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		parts := strings.SplitN(s, ".", 2)
		intPart, _ = strconv.ParseInt(parts[0], 10, 64)
		if len(parts) == 2 {
			fracStr = "." + parts[1]
		}
	}

	negative := intPart < 0
	if negative {
		intPart = -intPart
	}

	s := strconv.FormatInt(intPart, 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}

	out := string(result) + fracStr
	if negative {
		out = "-" + out
	}
	return out
}

// humanDuration converts a duration in seconds into a compact human-readable string.
// For example, 9000 seconds becomes "2h30m".
func humanDuration(v any) string {
	f, err := toFloat(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	total := int(math.Round(f))

	days := total / 86400
	total %= 86400
	hours := total / 3600
	total %= 3600
	minutes := total / 60
	seconds := total % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	return strings.Join(parts, "")
}
