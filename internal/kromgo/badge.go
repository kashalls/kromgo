package kromgo

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/golang/freetype/truetype"
	"github.com/home-operations/kromgo/internal/config"
	"golang.org/x/image/font"
)

const defaultBadgeFontSize = 11

// badgeRenderer draws shields-style SVG badges (an optional left icon, a label, and
// a value) entirely in-process. Text widths are measured with the configured font;
// a sync.Pool guards the font.Face, which is not safe for concurrent use.
type badgeRenderer struct {
	size  float64
	faces sync.Pool
}

// newBadgeRenderer parses the configured font and returns a renderer.
func newBadgeRenderer(cfg config.BadgeDefaults) (*badgeRenderer, error) {
	data, err := resolveBadgeFont(cfg.Font)
	if err != nil {
		return nil, fmt.Errorf("badge font: %w", err)
	}
	f, err := truetype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing badge font: %w", err)
	}
	size := float64(cfg.Size)
	if size <= 0 {
		size = defaultBadgeFontSize
	}
	return &badgeRenderer{
		size: size,
		faces: sync.Pool{New: func() any {
			return truetype.NewFace(f, &truetype.Options{Size: size, DPI: 72, Hinting: font.HintingFull})
		}},
	}, nil
}

func (b *badgeRenderer) measure(s string) int {
	face := b.faces.Get().(font.Face)
	defer b.faces.Put(face)
	return font.MeasureString(face, s).Round()
}

// render produces an SVG badge. iconPath is 24x24 SVG path data (or ""), label is
// the left text (may be ""), message the right text, color a name or hex.
func (b *badgeRenderer) render(style, iconPath, label, message, color string) []byte {
	const (
		xPad    = 6 // horizontal padding around each text segment
		iconX   = 5 // icon left margin
		iconGap = 3 // gap between icon and label text
	)
	size := int(b.size)
	h := size + 9
	iconSize := h - 6
	hasIcon := iconPath != ""
	hasLabel := label != ""

	labelLeft := xPad
	if hasIcon {
		labelLeft = iconX + iconSize + iconGap
	}

	var labelSeg int
	switch {
	case hasLabel:
		labelSeg = labelLeft + b.measure(label) + xPad
	case hasIcon:
		labelSeg = iconX + iconSize + xPad // icon-only label side
	}
	msgW := b.measure(message)
	msgSeg := msgW + 2*xPad
	total := labelSeg + msgSeg

	rx := 3
	gradient := true
	if style == config.StyleFlatSquare {
		rx, gradient = 0, false
	}
	baseline := (h+size)/2 - 1

	var s strings.Builder
	fmt.Fprintf(&s, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" role="img">`, total, h)
	if gradient {
		s.WriteString(`<linearGradient id="g" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`)
	}
	fmt.Fprintf(&s, `<clipPath id="r"><rect width="%d" height="%d" rx="%d" fill="#fff"/></clipPath>`, total, h, rx)
	s.WriteString(`<g clip-path="url(#r)">`)
	fmt.Fprintf(&s, `<rect width="%d" height="%d" fill="#555"/>`, labelSeg, h)
	fmt.Fprintf(&s, `<rect x="%d" width="%d" height="%d" fill="%s"/>`, labelSeg, msgSeg, h, colorNameToHex(color))
	if gradient {
		fmt.Fprintf(&s, `<rect width="%d" height="%d" fill="url(#g)"/>`, total, h)
	}
	s.WriteString(`</g>`)

	if hasIcon {
		// iconPath is static, trusted registry data (not user input).
		scale := float64(iconSize) / 24.0
		fmt.Fprintf(&s, `<g transform="translate(%d %d) scale(%.4f)"><path fill="#fff" d="%s"/></g>`,
			iconX, (h-iconSize)/2, scale, iconPath)
	}

	if hasLabel || message != "" {
		fmt.Fprintf(&s, `<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="%d">`, size)
		if hasLabel {
			writeBadgeText(&s, labelLeft+b.measure(label)/2, baseline, label)
		}
		writeBadgeText(&s, labelSeg+xPad+msgW/2, baseline, message)
		s.WriteString(`</g>`)
	}
	s.WriteString(`</svg>`)
	return []byte(s.String())
}

// writeBadgeText writes centered text with a subtle drop shadow. The text is
// HTML-escaped — message/label can derive from metric labels (untrusted).
func writeBadgeText(s *strings.Builder, x, y int, text string) {
	esc := html.EscapeString(text)
	fmt.Fprintf(s, `<text x="%d" y="%d" fill="#010101" fill-opacity=".3">%s</text>`, x, y+1, esc)
	fmt.Fprintf(s, `<text x="%d" y="%d">%s</text>`, x, y, esc)
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

const (
	colorBlue  = "#007ec6"
	colorGreen = "#97ca00"
	colorGrey  = "#9f9f9f"
)

// badgeColors maps shields.io color names to hex. "" is the default (blue).
var badgeColors = map[string]string{
	"":              colorBlue,
	"blue":          colorBlue,
	"brightgreen":   "#4c1",
	"green":         colorGreen,
	"yellow":        "#dfb317",
	"yellowgreen":   "#a4a61d",
	"orange":        "#fe7d37",
	"red":           "#e05d44",
	"grey":          "#555",
	"gray":          "#555",
	"lightgrey":     colorGrey,
	"lightgray":     colorGrey,
	"success":       colorGreen,
	"important":     "#fe7d37",
	"critical":      "#e05d44",
	"informational": colorBlue,
	"inactive":      colorGrey,
}

// colorNameToHex maps a shields.io color name (or a hex string) to a hex color,
// falling back to green for an unknown name or malformed hex.
func colorNameToHex(color string) string {
	if strings.HasPrefix(color, "#") {
		if hexColorRe.MatchString(color) {
			return color
		}
		return colorGreen
	}
	if hex, ok := badgeColors[color]; ok {
		return hex
	}
	return colorGreen
}

// resolveIcon parses an "mdi:<name>" reference into 24x24 SVG path data. Empty
// input returns "". An unknown name or non-mdi prefix errors (fails fast at startup).
func resolveIcon(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	name, ok := strings.CutPrefix(ref, "mdi:")
	if !ok {
		return "", fmt.Errorf("icon %q: only mdi: icons are supported (e.g. mdi:server-outline)", ref)
	}
	path, ok := mdiIcons[name]
	if !ok {
		return "", fmt.Errorf("unknown icon %q", ref)
	}
	return path, nil
}
