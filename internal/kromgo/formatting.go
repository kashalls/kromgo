package kromgo

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The functions below are registered with the CEL environment (see expr.go) and
// callable from a metric's valueExpr/colorExpr, e.g. humanizeBytes(result). They take
// the (double) sample value and return a display string. All formatting is hand-rolled
// (no external humanize dependency) so the output is exactly what kromgo specifies.

// byteUnits is the SI suffix ladder for humanizeBytes (decimal, powers of 1000).
var byteUnits = [...]string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

// humanizeBytes formats a byte count with SI decimal units and no space, scaling by
// powers of 1000: 1500000 -> "1.5MB", 1000 -> "1kB", 512 -> "512B". Scaled values keep
// at most one decimal, with a trailing ".0" stripped.
func humanizeBytes(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return humanizeFloat(f) // "+Inf" / "-Inf" / "NaN", not "+InfEB" / "NaNB"
	}
	v, i := f, 0
	for math.Abs(v) >= 1000 && i < len(byteUnits)-1 {
		v /= 1000
		i++
	}
	// Rounding to one decimal can tip |v| up to 1000 (e.g. 999999 → 999.999); carry
	// into the next unit so it renders "1MB", not "1000kB".
	if math.Round(math.Abs(v)*10)/10 >= 1000 && i < len(byteUnits)-1 {
		v /= 1000
		i++
	}
	return trimOneDecimal(v) + byteUnits[i]
}

// humanizeCommas formats a number with comma thousands separators in the integer part,
// e.g. 157121 -> "157,121", 1234.56 -> "1,234.56", -1234 -> "-1,234".
func humanizeCommas(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return humanizeFloat(f) // "+Inf" / "-Inf" / "NaN", not a comma-mangled "+,Inf"
	}
	s := humanizeFloat(f)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac, hasFrac := strings.Cut(s, ".")

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i := 0; i < len(intPart); i++ {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(intPart[i])
	}
	if hasFrac {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return b.String()
}

// humanizeFloat formats a float as a plain decimal with trailing zeros stripped, e.g.
// 200.0 -> "200", 2.50 -> "2.5". No separators, units, or scientific notation.
func humanizeFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// trimOneDecimal formats v to a single decimal place, then strips a trailing ".0"
// (1.5 -> "1.5", 1.0 -> "1"). Used for the scaled byte values.
func trimOneDecimal(v float64) string {
	return strings.TrimSuffix(strconv.FormatFloat(v, 'f', 1, 64), ".0")
}

// durationUnits is the suffix ladder for humanizeDuration, largest first. Months
// use 30 days and years 365 (approximate, which is fine for a badge). Months are
// "mo", not "m", so they never collide with minutes in the same string.
var durationUnits = []struct {
	secs   int
	suffix string
}{
	{365 * 86400, "y"},
	{30 * 86400, "mo"},
	{86400, "d"},
	{3600, "h"},
	{60, "m"},
	{1, "s"},
}

// humanizeDuration formats a number of seconds as a compact, magnitude-adaptive
// span: it emits the up-to-three most-significant non-zero units, so one function
// reads well from seconds to years — 90 -> "1m30s", 9000 -> "2h30m",
// 40348800 -> "1y3mo12d". Negative input and anything that rounds to zero render
// as "0s".
func humanizeDuration(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return humanizeFloat(f) // "+Inf" / "-Inf" / "NaN" — int(NaN/Inf) is undefined in Go
	}
	rem := int(math.Round(f))
	if rem <= 0 {
		return "0s"
	}
	var b strings.Builder
	parts := 0
	for _, u := range durationUnits {
		if parts == 3 {
			break
		}
		n := rem / u.secs
		if n == 0 {
			continue
		}
		fmt.Fprintf(&b, "%d%s", n, u.suffix)
		rem -= n * u.secs
		parts++
	}
	return b.String()
}

// humanizeDurationDays formats a number of seconds as a whole-day count, e.g.
// 5961600 -> "69d". Unlike humanizeDuration it never rolls up to months/years —
// just total days, truncated and clamped at zero.
func humanizeDurationDays(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return humanizeFloat(f) // "+Inf" / "-Inf" / "NaN" — int(NaN/Inf) is undefined in Go
	}
	days := int(f / 86400)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("%dd", days)
}
