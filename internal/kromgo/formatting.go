package kromgo

import (
	"fmt"
	"math"
	"strings"

	"github.com/dustin/go-humanize"
)

// The functions below are registered with the CEL environment (see expr.go) and
// callable from a metric's value/color expression, e.g. humanizeBytes(result).
// They take the (double) sample value. Byte and number formatting is delegated
// to github.com/dustin/go-humanize; humanizeDuration is kept custom so it stays
// compact and badge-friendly.

// humanizeBytes formats a byte count with IEC binary units, e.g. 1572864 -> "1.5 MiB".
func humanizeBytes(f float64) string { return humanize.IBytes(uint64(f)) }

// humanizeSIBytes formats a byte count with SI decimal units, e.g. 1500000 -> "1.5 MB".
func humanizeSIBytes(f float64) string { return humanize.Bytes(uint64(f)) }

// humanizeNumber adds comma thousands separators, e.g. 157121 -> "157,121".
func humanizeNumber(f float64) string { return humanize.Commaf(f) }

// humanizeFloat formats a float as plain decimal with trailing zeros stripped,
// e.g. 200.0 -> "200", 2.50 -> "2.5". Unlike humanizeNumber it adds no
// separators — handy when CEL's own string(result) would use scientific
// notation or keep noisy zeros.
func humanizeFloat(f float64) string { return humanize.Ftoa(f) }

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

// humanizeDays formats a number of seconds as a whole-day count, e.g. 5961600 -> "69d".
// Unlike humanizeDuration it never rolls up to months/years — just total days.
func humanizeDays(f float64) string {
	days := int(f / 86400)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("%dd", days)
}
