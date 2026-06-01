package kromgo

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strings"
	"sync"
)

// mdiData is the full Material Design Icons set (https://pictogrammers.com/library/mdi/,
// Apache-2.0): a gzipped, tab-separated "name\tpath" table, one 24x24 glyph path per
// line. siData is the same for the Simple Icons brand set (https://simpleicons.org/,
// CC0-1.0). Both are built from npm packages (@mdi/svg, simple-icons) by cmd/genassets
// (not committed) and embedded; see the README's "Building from source".
//
//go:embed assets/mdi.txt.gz
var mdiData []byte

//go:embed assets/si.txt.gz
var siData []byte

// mdiIcons / siIcons lazily decode their table into a name→path map on first use, so
// configs that don't use a set never pay its decode cost. The sets are read-only after build.
var (
	mdiIcons = sync.OnceValue(func() map[string]string { return decodeIcons(mdiData) })
	siIcons  = sync.OnceValue(func() map[string]string { return decodeIcons(siData) })
)

// decodeIcons parses a gzipped, tab-separated "name\tpath" table into a name→path map.
func decodeIcons(data []byte) map[string]string {
	icons := map[string]string{}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return icons // embedded data is generated; a failure here just means "unknown icon"
	}
	defer func() { _ = zr.Close() }()

	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 64*1024), 256*1024) // a few glyph paths exceed the 64K default
	for sc.Scan() {
		if name, path, ok := strings.Cut(sc.Text(), "\t"); ok {
			icons[name] = path
		}
	}
	return icons
}
