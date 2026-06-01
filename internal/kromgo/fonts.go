package kromgo

import (
	"fmt"
	"os"

	charts "github.com/go-analyze/charts"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// Graph fonts: the go-analyze/charts built-ins (always available) plus the Go font
// family embedded from golang.org/x/image (compiled into the binary — no base-image
// files). A graph's font may also be a path to a .ttf on disk.

// builtinGraphFonts are the chart library's bundled fonts, selectable by name.
var builtinGraphFonts = map[string]bool{
	charts.FontFamilyRoboto:       true,
	charts.FontFamilyNotoSans:     true,
	charts.FontFamilyNotoSansBold: true,
}

// embeddedGraphFonts are extra faces embedded from x/image, keyed by their ?font name.
var embeddedGraphFonts = map[string][]byte{
	"go-regular": goregular.TTF,
	"go-bold":    gobold.TTF,
	"go-medium":  gomedium.TTF,
	"go-mono":    gomono.TTF,
}

// resolveGraphFont resolves a font name to a parsed font: a built-in or embedded
// registry name, or a path to a .ttf on disk. Empty returns nil (the library default,
// Roboto). It is called once per graph at startup.
func resolveGraphFont(name string) (*truetype.Font, error) {
	switch {
	case name == "":
		return nil, nil
	case builtinGraphFonts[name]:
		if f := charts.GetFont(name); f != nil {
			return f, nil
		}
		return nil, fmt.Errorf("font %q unavailable", name)
	case embeddedGraphFonts[name] != nil:
		return truetype.Parse(embeddedGraphFonts[name])
	default:
		data, err := os.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading font %q (not a known font name either): %w", name, err)
		}
		return truetype.Parse(data)
	}
}
