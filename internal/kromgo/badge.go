package kromgo

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/home-operations/kromgo/internal/config"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const defaultBadgeFontSize = 11

// badgeRenderer draws shields-style SVG badges (an optional left icon, a label, and
// a value). Text is rendered as vector paths from the configured font, so the badge
// is identical in every viewer (no dependence on system fonts or SVG textLength) and
// segment widths always match the glyphs exactly. sfnt.Font is safe for concurrent
// use as long as each call uses its own Buffer.
type badgeRenderer struct {
	font *sfnt.Font
	size float64
}

// newBadgeRenderer parses the configured font and returns a renderer.
func newBadgeRenderer(cfg config.BadgeDefaults) (*badgeRenderer, error) {
	data, err := resolveBadgeFont(cfg.Font)
	if err != nil {
		return nil, fmt.Errorf("badge font: %w", err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing badge font: %w", err)
	}
	size := float64(cfg.Size)
	if size <= 0 {
		size = defaultBadgeFontSize
	}
	return &badgeRenderer{font: f, size: size}, nil
}

func (b *badgeRenderer) ppem() fixed.Int26_6 { return fixed.Int26_6(b.size * 64) }

// measure returns the rendered width of s in pixels (sum of glyph advances).
func (b *badgeRenderer) measure(s string) int {
	var buf sfnt.Buffer
	var w fixed.Int26_6
	for _, r := range s {
		gid, err := b.font.GlyphIndex(&buf, r)
		if err != nil || gid == 0 {
			w += fixed.Int26_6(b.size * 0.5 * 64) // fallback for a missing glyph
			continue
		}
		if adv, err := b.font.GlyphAdvance(&buf, gid, b.ppem(), font.HintingNone); err == nil {
			w += adv
		}
	}
	return int(float64(w)/64 + 0.5)
}

// glyphPath returns SVG path data for s, with the text baseline at (originX, baseline).
func (b *badgeRenderer) glyphPath(s string, originX, baseline float64) string {
	var buf sfnt.Buffer
	pen := originX
	var d strings.Builder
	f26 := func(v fixed.Int26_6) float64 { return float64(v) / 64 }

	for _, r := range s {
		gid, err := b.font.GlyphIndex(&buf, r)
		if err == nil && gid != 0 {
			if segs, err := b.font.LoadGlyph(&buf, gid, b.ppem(), nil); err == nil {
				for i, seg := range segs {
					a := seg.Args
					switch seg.Op {
					case sfnt.SegmentOpMoveTo:
						if i > 0 {
							d.WriteByte('Z')
						}
						fmt.Fprintf(&d, "M%.1f %.1f", pen+f26(a[0].X), baseline+f26(a[0].Y))
					case sfnt.SegmentOpLineTo:
						fmt.Fprintf(&d, "L%.1f %.1f", pen+f26(a[0].X), baseline+f26(a[0].Y))
					case sfnt.SegmentOpQuadTo:
						fmt.Fprintf(&d, "Q%.1f %.1f %.1f %.1f", pen+f26(a[0].X), baseline+f26(a[0].Y), pen+f26(a[1].X), baseline+f26(a[1].Y))
					case sfnt.SegmentOpCubeTo:
						fmt.Fprintf(&d, "C%.1f %.1f %.1f %.1f %.1f %.1f", pen+f26(a[0].X), baseline+f26(a[0].Y), pen+f26(a[1].X), baseline+f26(a[1].Y), pen+f26(a[2].X), baseline+f26(a[2].Y))
					}
				}
				if len(segs) > 0 {
					d.WriteByte('Z')
				}
			}
		}
		if adv, err := b.font.GlyphAdvance(&buf, gid, b.ppem(), font.HintingNone); err == nil {
			pen += f26(adv)
		} else {
			pen += b.size * 0.5
		}
	}
	return d.String()
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

	labelW := 0
	if hasLabel {
		labelW = b.measure(label)
	}
	var labelSeg int
	switch {
	case hasLabel:
		labelSeg = labelLeft + labelW + xPad
	case hasIcon:
		labelSeg = iconX + iconSize + xPad // icon-only label side
	}
	msgW := b.measure(message)
	msgSeg := msgW + 2*xPad
	total := labelSeg + msgSeg

	rx, gradStops := styleAppearance(style)
	baseline := (h+size)/2 - 1

	var s strings.Builder
	fmt.Fprintf(&s, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`, total, h, total, h)
	if gradStops != "" {
		fmt.Fprintf(&s, `<linearGradient id="g" x2="0" y2="100%%">%s</linearGradient>`, gradStops)
	}
	fmt.Fprintf(&s, `<clipPath id="r"><rect width="%d" height="%d" rx="%d" fill="#fff"/></clipPath>`, total, h, rx)
	// Everything is drawn inside the clip group so the rounded corners trim every
	// element — and any stray glyph ink can never escape the badge bounds.
	s.WriteString(`<g clip-path="url(#r)">`)
	fmt.Fprintf(&s, `<rect width="%d" height="%d" fill="#555"/>`, labelSeg, h)
	fmt.Fprintf(&s, `<rect x="%d" width="%d" height="%d" fill="%s"/>`, labelSeg, msgSeg, h, colorNameToHex(color))
	if gradStops != "" {
		fmt.Fprintf(&s, `<rect width="%d" height="%d" fill="url(#g)"/>`, total, h)
	}
	if hasIcon {
		// iconPath is static, trusted registry data (not user input).
		scale := float64(iconSize) / 24.0
		fmt.Fprintf(&s, `<g transform="translate(%d %d) scale(%.4f)"><path fill="#fff" d="%s"/></g>`,
			iconX, (h-iconSize)/2, scale, iconPath)
	}

	// Text as vector paths from the font: exact widths, no system-font/textLength
	// dependency. Build both segments' paths, then a grey shadow + a white copy.
	// Untrusted text becomes path geometry, so it can't inject markup.
	var paths strings.Builder
	bl := float64(baseline)
	if hasLabel {
		paths.WriteString(b.glyphPath(label, float64(labelLeft), bl))
	}
	paths.WriteString(b.glyphPath(message, float64(labelSeg+xPad), bl))
	if d := paths.String(); d != "" {
		fmt.Fprintf(&s, `<path transform="translate(0 1)" fill="#010101" fill-opacity=".3" d="%s"/>`, d)
		fmt.Fprintf(&s, `<path fill="#fff" d="%s"/>`, d)
	}
	s.WriteString(`</g></svg>`)
	return []byte(s.String())
}

// styleAppearance returns the corner radius and linear-gradient stops for a badge
// style. flat-square has square corners and no gloss; plastic gets a pronounced
// glossy gradient; flat (default) gets a subtle darkening overlay.
func styleAppearance(style string) (rx int, gradStops string) {
	switch style {
	case config.StyleFlatSquare:
		return 0, ""
	case config.StylePlastic:
		return 4, `<stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-color="#000" stop-opacity=".3"/><stop offset="1" stop-color="#000" stop-opacity=".5"/>`
	default: // flat
		return 3, `<stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/>`
	}
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

// iconSets maps an icon-set prefix to its lazily-decoded name→path table.
var iconSets = map[string]func() map[string]string{
	"mdi": mdiIcons, // Material Design Icons (https://pictogrammers.com/library/mdi/)
	"si":  siIcons,  // Simple Icons brand set (https://simpleicons.org/)
}

// resolveIcon parses a "<set>:<name>" reference (e.g. "mdi:server-outline" or
// "si:kubernetes") into 24x24 SVG path data. Empty input returns "". An unknown set
// or name errors (fails fast at startup).
func resolveIcon(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	prefix, name, ok := strings.Cut(ref, ":")
	if !ok {
		return "", fmt.Errorf("icon %q: expected \"<set>:<name>\" (e.g. mdi:server-outline or si:kubernetes)", ref)
	}
	set, ok := iconSets[prefix]
	if !ok {
		return "", fmt.Errorf("icon %q: unknown icon set %q (supported: mdi, si)", ref, prefix)
	}
	path, ok := set()[name]
	if !ok {
		return "", fmt.Errorf("unknown icon %q", ref)
	}
	return path, nil
}
