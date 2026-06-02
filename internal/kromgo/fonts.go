package kromgo

import (
	_ "embed"
	"fmt"

	"github.com/golang/freetype/truetype"
)

// Fonts are compiled into the binary, never read from disk — the image is scratch
// and we control the set. DejaVu Sans is the default for badges and graphs (the free,
// metric-compatible stand-in for the Verdana shields.io renders with); Comic Neue is a
// second selectable face. Both are vendored via npm and generated into assets/ by
// cmd/genassets (regular + bold). Add a face by vendoring it (npm) and PRing it here.

//go:embed assets/dejavu-sans.ttf
var dejavuSansTTF []byte

//go:embed assets/dejavu-sans-bold.ttf
var dejavuSansBoldTTF []byte

//go:embed assets/comic-neue.ttf
var comicNeueTTF []byte

//go:embed assets/comic-neue-bold.ttf
var comicNeueBoldTTF []byte

var embeddedFonts = map[string][]byte{
	"dejavu-sans":      dejavuSansTTF,
	"dejavu-sans-bold": dejavuSansBoldTTF,
	"comic-neue":       comicNeueTTF,
	"comic-neue-bold":  comicNeueBoldTTF,
}

// resolveBadgeFont returns the TTF bytes for a badge font name (empty = the default
// DejaVu Sans face). The badge renderer parses these bytes with sfnt to draw glyph paths.
func resolveBadgeFont(name string) ([]byte, error) {
	if name == "" {
		return dejavuSansTTF, nil
	}
	if data := embeddedFonts[name]; data != nil {
		return data, nil
	}
	return nil, fmt.Errorf("unknown font %q", name)
}

// resolveGraphFont returns the parsed font for a graph font name (empty = the default
// DejaVu Sans face). Graphs render their text through the chart library with this font.
func resolveGraphFont(name string) (*truetype.Font, error) {
	if name == "" {
		return truetype.Parse(dejavuSansTTF)
	}
	if data := embeddedFonts[name]; data != nil {
		return truetype.Parse(data)
	}
	return nil, fmt.Errorf("unknown font %q", name)
}
