package kromgo

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/essentialkaos/go-badge"
	"github.com/home-operations/kromgo/internal/config"
	"golang.org/x/image/font/gofont/goregular"
)

const defaultBadgeFontSize = 11

// badgePool produces *badge.Generator instances. go-badge's Generator wraps a
// *font.Drawer, which is not safe for concurrent use, so each request borrows one.
type badgePool struct{ pool sync.Pool }

// newBadgePool validates the configured (or embedded default) font and returns a pool.
func newBadgePool(cfg config.BadgeDefaults) (*badgePool, error) {
	size := cfg.Size
	if size <= 0 {
		size = defaultBadgeFontSize
	}

	fontData := goregular.TTF
	if cfg.Font != "" {
		data, err := os.ReadFile(cfg.Font)
		if err != nil {
			return nil, fmt.Errorf("reading badge font: %w", err)
		}
		fontData = data
	}

	// Validate the font up front so a bad font fails at startup, not per request.
	if _, err := badge.NewGeneratorFromBytes(fontData, size); err != nil {
		return nil, fmt.Errorf("loading badge font: %w", err)
	}

	return &badgePool{pool: sync.Pool{New: func() any {
		gen, err := badge.NewGeneratorFromBytes(fontData, size)
		if err != nil {
			// Unreachable: the same fontData/size was validated in newBadgePool.
			panic(fmt.Errorf("badge generator: %w", err))
		}
		return gen
	}}}, nil
}

// write renders an SVG badge for the given style and writes it to w.
func (b *badgePool) write(w http.ResponseWriter, style, title, message, color string) {
	gen := b.pool.Get().(*badge.Generator)
	defer b.pool.Put(gen)

	// Escape: go-badge writes the title/message into SVG <text> without escaping,
	// so a CEL value or metric label could otherwise inject markup/script.
	title = html.EscapeString(title)
	message = html.EscapeString(message)

	hex := colorNameToHex(color)
	var svg []byte
	switch style {
	case "plastic":
		svg = gen.GeneratePlastic(title, message, hex)
	case "flat-square":
		svg = gen.GenerateFlatSquare(title, message, hex)
	default:
		svg = gen.GenerateFlat(title, message, hex)
	}
	writeSVG(w, svg)
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

// colorNameToHex maps a shields.io color name (or hex string) to a go-badge hex color.
func colorNameToHex(colorName string) string {
	if strings.HasPrefix(colorName, "#") {
		if hexColorRe.MatchString(colorName) {
			return colorName
		}
		return badge.COLOR_GREEN
	}

	switch colorName {
	case "", "blue":
		return badge.COLOR_BLUE
	case "brightgreen":
		return badge.COLOR_BRIGHTGREEN
	case "green":
		return badge.COLOR_GREEN
	case "grey":
		return badge.COLOR_GREY
	case "lightgrey":
		return badge.COLOR_LIGHTGREY
	case "orange":
		return badge.COLOR_ORANGE
	case "red":
		return badge.COLOR_RED
	case "yellow":
		return badge.COLOR_YELLOW
	case "yellowgreen":
		return badge.COLOR_YELLOWGREEN
	case "success":
		return badge.COLOR_SUCCESS
	case "important":
		return badge.COLOR_IMPORTANT
	case "critical":
		return badge.COLOR_CRITICAL
	case "informational":
		return badge.COLOR_INFORMATIONAL
	case "inactive":
		return badge.COLOR_INACTIVE
	default:
		return badge.COLOR_GREEN
	}
}
