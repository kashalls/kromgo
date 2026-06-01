package kromgo

import (
	"fmt"
	"math"
	"strings"

	"github.com/dustin/go-humanize"
)

// The functions below are registered with the CEL environment (see expr.go) and
// callable from a metric's value/color expression, e.g. humanBytes(result). They
// take the (double) sample value. Byte and thousands formatting is delegated to
// github.com/dustin/go-humanize; duration/age are kept compact and badge-friendly.

// humanBytes formats a byte count with IEC binary units, e.g. 1572864 -> "1.5 MiB".
func humanBytes(f float64) string { return humanize.IBytes(uint64(f)) }

// humanSIBytes formats a byte count with SI decimal units, e.g. 1500000 -> "1.5 MB".
func humanSIBytes(f float64) string { return humanize.Bytes(uint64(f)) }

// humanizeThousands adds comma thousands separators, e.g. 157121 -> "157,121".
func humanizeThousands(f float64) string { return humanize.Commaf(f) }

// humanizeFtoa formats a float as plain decimal with trailing zeros stripped,
// e.g. 200.0 -> "200", 2.50 -> "2.5". Unlike humanizeThousands it adds no
// separators — handy when CEL's own string(result) would use scientific
// notation or keep noisy zeros.
func humanizeFtoa(f float64) string { return humanize.Ftoa(f) }

// humanDuration formats a number of seconds as a compact duration, e.g. 9000 -> "2h30m".
func humanDuration(f float64) string {
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

// humanizeAge formats a number of seconds as a coarse age — years, months,
// days, no hours/minutes/seconds — e.g. ~467d -> "1y3m12d". Takes seconds (like
// humanDuration) so it drops straight onto a `time() - created` query; it just
// renders at a coarser resolution. Months use 30 days and years 365
// (approximate, which is fine for an "age" badge). Zero components are omitted;
// anything under a day rounds down to "0d".
func humanizeAge(f float64) string {
	days := int(f / 86400)
	if days < 0 {
		days = 0
	}
	years := days / 365
	days %= 365
	months := days / 30
	days %= 30

	var b strings.Builder
	if years > 0 {
		fmt.Fprintf(&b, "%dy", years)
	}
	if months > 0 {
		fmt.Fprintf(&b, "%dm", months)
	}
	if days > 0 || b.Len() == 0 {
		fmt.Fprintf(&b, "%dd", days)
	}
	return b.String()
}
