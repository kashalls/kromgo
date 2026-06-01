package kromgo

import (
	"fmt"

	charts "github.com/go-analyze/charts"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// Fonts are compiled into the binary, never read from disk — the image is scratch
// and we control the set. Add a face by importing it here and PRing it into the
// registry. embeddedFonts are the Go family from golang.org/x/image; graphs can
// also use the chart library's bundled fonts (builtinGraphFonts).

var embeddedFonts = map[string][]byte{
	"go-regular": goregular.TTF,
	"go-bold":    gobold.TTF,
	"go-medium":  gomedium.TTF,
	"go-mono":    gomono.TTF,
}

// builtinGraphFonts are the chart library's bundled fonts, selectable by name.
var builtinGraphFonts = map[string]bool{
	charts.FontFamilyRoboto:       true,
	charts.FontFamilyNotoSans:     true,
	charts.FontFamilyNotoSansBold: true,
}

// resolveBadgeFont returns the TTF bytes for a badge font name (empty = the default
// Go regular face). Badges render through go-badge, which needs the raw bytes.
func resolveBadgeFont(name string) ([]byte, error) {
	if name == "" {
		return goregular.TTF, nil
	}
	if data := embeddedFonts[name]; data != nil {
		return data, nil
	}
	return nil, fmt.Errorf("unknown font %q", name)
}

// resolveGraphFont returns the parsed font for a graph font name: a chart-library
// built-in (roboto/notosans/…) or an embedded Go face. Empty returns nil (the
// library default, Roboto).
func resolveGraphFont(name string) (*truetype.Font, error) {
	switch {
	case name == "":
		return nil, nil
	case builtinGraphFonts[name]:
		if f := charts.GetFont(name); f != nil {
			return f, nil
		}
		return nil, fmt.Errorf("font %q unavailable", name)
	case embeddedFonts[name] != nil:
		return truetype.Parse(embeddedFonts[name])
	default:
		return nil, fmt.Errorf("unknown font %q", name)
	}
}
