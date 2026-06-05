package kromgo

import (
	"cmp"
	"fmt"
	"html"
	"regexp"
	"strconv"
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

// badgeSpec is the fully-resolved input to render: the style, an optional left icon
// (24x24 SVG path data), the left label and right message text, their background
// colors (name or hex; labelColor "" = the default grey), and the badge id, which
// namespaces the SVG's element ids so inlined badges don't collide.
type badgeSpec struct {
	style      string
	iconPath   string
	label      string
	message    string
	color      string
	labelColor string
	id         string
}

// render produces an SVG badge from a fully-resolved spec.
func (b *badgeRenderer) render(spec badgeSpec) []byte {
	const (
		xPad    = 6 // horizontal padding around each text segment
		iconX   = 5 // icon left margin
		iconGap = 3 // gap between icon and label text
	)
	size := int(b.size)
	h := size + 9
	iconSize := h - 6
	hasIcon := spec.iconPath != ""
	hasLabel := spec.label != ""
	// With an icon but no label, the icon rides on the message segment and no label
	// segment is drawn at all — a single-color badge, mirroring shields.io's
	// empty-label form (just a logo and a value, no separate grey box).
	iconOnMessage := hasIcon && !hasLabel

	// Label segment: drawn only when there's label text. An icon sharing it shifts
	// the label text right to clear the glyph.
	labelLeft := xPad
	if hasIcon && hasLabel {
		labelLeft = iconX + iconSize + iconGap
	}
	labelSeg := 0
	if hasLabel {
		labelSeg = labelLeft + b.measure(spec.label) + xPad
	}

	// Message segment: starts after the label segment. When the icon rides here
	// instead, the text is pushed right to clear it and the icon's width joins this
	// segment.
	msgW := b.measure(spec.message)
	msgLeft := labelSeg + xPad
	msgSeg := msgW + 2*xPad
	if iconOnMessage {
		msgLeft = iconX + iconSize + iconGap
		msgSeg = msgLeft + msgW + xPad
	}
	total := labelSeg + msgSeg

	rx, gradStops := styleAppearance(spec.style)
	baseline := (h+size)/2 - 1

	msgHex := colorNameToHex(spec.color)
	labelHex := cmp.Or(spec.labelColor, labelBg)
	// alt is the badge's accessible name. It's the one place untrusted label/message
	// text becomes markup rather than path geometry, so it must be XML-escaped.
	alt := html.EscapeString(accessibleText(spec.label, spec.message))
	// Namespace the gradient/clip ids by badge id: SVG id resolution is document-global,
	// so two kromgo SVGs inlined in one HTML page would otherwise have the second's
	// url(#…) refs resolve to the first's gradient/clip.
	gradID := "g-" + xmlIDSafe(spec.id)
	clipID := "r-" + xmlIDSafe(spec.id)

	var s strings.Builder
	// role="img" + aria-label make the badge a single labelled image for assistive
	// tech (the glyph-path text is otherwise invisible to it); <title> adds a native
	// hover tooltip.
	fmt.Fprintf(&s, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">`, total, h, total, h, alt)
	fmt.Fprintf(&s, `<title>%s</title>`, alt)
	if gradStops != "" {
		fmt.Fprintf(&s, `<linearGradient id="%s" x2="0" y2="100%%">%s</linearGradient>`, gradID, gradStops)
	}
	fmt.Fprintf(&s, `<clipPath id="%s"><rect width="%d" height="%d" rx="%d" fill="#fff"/></clipPath>`, clipID, total, h, rx)
	// Everything is drawn inside the clip group so the rounded corners trim every
	// element — and any stray glyph ink can never escape the badge bounds.
	fmt.Fprintf(&s, `<g clip-path="url(#%s)">`, clipID)
	if labelSeg > 0 {
		fmt.Fprintf(&s, `<rect width="%d" height="%d" fill="%s"/>`, labelSeg, h, labelHex)
	}
	fmt.Fprintf(&s, `<rect x="%d" width="%d" height="%d" fill="%s"/>`, labelSeg, msgSeg, h, msgHex)
	if gradStops != "" {
		fmt.Fprintf(&s, `<rect width="%d" height="%d" fill="url(#%s)"/>`, total, h, gradID)
	}
	if hasIcon {
		// iconPath is static, trusted registry data (not user input). It takes a fill
		// legible on whichever segment it sits on: the label segment, or the message
		// segment when there's no label.
		iconBg := labelHex
		if iconOnMessage {
			iconBg = msgHex
		}
		iconColor, _ := colorsForBackground(iconBg)
		scale := float64(iconSize) / 24.0
		fmt.Fprintf(&s, `<g transform="translate(%d %d) scale(%.4f)"><path fill="%s" d="%s"/></g>`,
			iconX, (h-iconSize)/2, scale, iconColor, spec.iconPath)
	}

	// Text as vector paths from the font: exact widths, no system-font/textLength
	// dependency, and untrusted text becomes geometry that can't inject markup. Each
	// segment's text + drop shadow take a color legible on that segment's background.
	bl := float64(baseline)
	if hasLabel {
		b.writeText(&s, spec.label, float64(labelLeft), bl, labelHex)
	}
	b.writeText(&s, spec.message, float64(msgLeft), bl, msgHex)
	s.WriteString(`</g></svg>`)
	return []byte(s.String())
}

// renderError draws a self-describing error badge — the id as label and a short
// reason as message — colored red for client errors (4xx) and grey for server or
// upstream errors (5xx), so an <img> shows the failure instead of a broken image.
func (b *badgeRenderer) renderError(id, reason string, code int) []byte {
	color := "lightgrey"
	if code < 500 {
		color = "red"
	}
	return b.render(badgeSpec{style: config.StyleFlat, label: id, message: reason, color: color, id: id})
}

// xmlIDSafe maps any character outside [A-Za-z0-9_-] to '-' so a badge id can safely
// namespace SVG element ids. Badge ids are already URL-safe; this guards '.' (which is
// valid in an id) and any error-badge label passed through as an id.
func xmlIDSafe(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return '-'
		}
	}, s)
}

// writeText draws s as glyph paths at (originX, baseline) on a background of bgHex:
// a 1px drop shadow beneath a fill, both colored for legibility on that background.
// Nothing is written for empty text.
func (b *badgeRenderer) writeText(s *strings.Builder, text string, originX, baseline float64, bgHex string) {
	d := b.glyphPath(text, originX, baseline)
	if d == "" {
		return
	}
	textColor, shadowColor := colorsForBackground(bgHex)
	fmt.Fprintf(s, `<path transform="translate(0 1)" fill="%s" fill-opacity=".3" d="%s"/>`, shadowColor, d)
	fmt.Fprintf(s, `<path fill="%s" d="%s"/>`, textColor, d)
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
	labelBg    = "#555" // label-side (left) background; always dark, so its text is white
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

// accessibleText is the badge's screen-reader / tooltip label: "label: message", or
// just whichever side is present.
func accessibleText(label, message string) string {
	if label != "" && message != "" {
		return label + ": " + message
	}
	return label + message
}

// brightnessThreshold is shields.io's split (on a 0..1 perceived-brightness scale)
// between backgrounds dark enough for white text and light ones that need dark text.
const brightnessThreshold = 0.69

// colorsForBackground picks a legible text color and drop-shadow color for text on a
// background of the given hex, mirroring shields.io: dark backgrounds get white text
// with a near-black shadow; light ones get dark text with a light shadow. An
// unparseable color is treated as dark (white text).
func colorsForBackground(hex string) (text, shadow string) {
	if r, g, b, ok := parseHexRGB(hex); ok {
		// W3C perceived brightness, scaled to 0..1.
		brightness := (float64(r)*299 + float64(g)*587 + float64(b)*114) / (255 * 1000)
		if brightness > brightnessThreshold {
			return "#333", "#ccc"
		}
	}
	return "#fff", "#010101"
}

// parseHexRGB extracts 8-bit r, g, b from a #rgb, #rgba, #rrggbb, or #rrggbbaa string
// (any alpha is ignored). ok is false for any other form.
func parseHexRGB(hex string) (r, g, b uint8, ok bool) {
	h := strings.TrimPrefix(hex, "#")
	switch len(h) {
	case 3, 4: // #rgb / #rgba — each nibble is doubled (f -> ff)
		v, err := strconv.ParseUint(h[:3], 16, 16)
		if err != nil {
			return 0, 0, 0, false
		}
		return uint8((v>>8)&0xf) * 0x11, uint8((v>>4)&0xf) * 0x11, uint8(v&0xf) * 0x11, true
	case 6, 8: // #rrggbb / #rrggbbaa
		v, err := strconv.ParseUint(h[:6], 16, 32)
		if err != nil {
			return 0, 0, 0, false
		}
		return uint8(v >> 16), uint8(v >> 8), uint8(v), true
	}
	return 0, 0, 0, false
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
