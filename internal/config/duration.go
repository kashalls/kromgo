package config

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// durationUnitRe matches a numeric value followed by a custom unit (y or d).
var durationUnitRe = regexp.MustCompile(`(\d+(?:\.\d+)?)(y|d)`)

var durationMultipliers = map[string]time.Duration{
	"y": 365 * 24 * time.Hour,
	"d": 24 * time.Hour,
}

// ParseDuration extends time.ParseDuration with support for days (d) and years (y).
// Units can be combined in any order: "1y30d", "7d12h", "3d5y", "2d".
// A value of "0" means unlimited (only meaningful for maxDuration config).
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	var total time.Duration
	for _, m := range durationUnitRe.FindAllStringSubmatch(s, -1) {
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		total += time.Duration(float64(durationMultipliers[m[2]]) * n)
	}

	// Strip the custom-unit components in one pass; whatever remains is handed to
	// the stdlib parser (e.g. "7d12h" -> "12h").
	remaining := durationUnitRe.ReplaceAllString(s, "")
	if remaining != "" {
		d, err := time.ParseDuration(remaining)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		total += d
	}

	return total, nil
}
