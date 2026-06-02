// Command genassets builds the assets that internal/kromgo embeds with //go:embed,
// from the npm packages vendored in node_modules (marked, github-markdown-css,
// @mdi/svg, simple-icons, and the DejaVu Sans + Comic Neue TrueType faces). Versions are
// pinned in package.json / package-lock.json
// and kept current by Renovate; this program is a pure local transform — it does no
// network I/O. Run `npm ci` (or `mise run assets`) first to populate node_modules.
//
//	npm ci && go run ./cmd/genassets
//
// The embedded files are not committed; a fresh checkout / CI / Docker build
// regenerates them before compiling internal/kromgo.
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	nodeModules = "node_modules"
	assetsDir   = "internal/kromgo/assets"
)

var (
	// sourceMapRe drops a trailing //# sourceMappingURL comment so the embedded JS
	// references nothing external.
	sourceMapRe = regexp.MustCompile(`(?m)^\s*//[#@]\s*sourceMappingURL=.*$\n?`)
	// svgPathRe extracts the geometry from an icon glyph's single <path d="…">. The
	// word boundary skips id="…" so it matches only the path's d attribute.
	svgPathRe = regexp.MustCompile(`\bd="([^"]+)"`)
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genassets:", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat(nodeModules); err != nil {
		return fmt.Errorf("%s missing — run `npm ci` (or `mise run assets`) first: %w", nodeModules, err)
	}
	steps := []struct {
		out string
		gen func() ([]byte, error)
	}{
		{"marked.js", genMarked},
		{"github-markdown.css", genMarkdownCSS},
		{"mdi.txt.gz", genMDI},
		{"si.txt.gz", genSimpleIcons},
		{"dejavu-sans.ttf", genFont("dejavu-fonts-ttf/ttf/DejaVuSans.ttf")},
		{"dejavu-sans-bold.ttf", genFont("dejavu-fonts-ttf/ttf/DejaVuSans-Bold.ttf")},
		{"comic-neue.ttf", genFont("@expo-google-fonts/comic-neue/400Regular/ComicNeue_400Regular.ttf")},
		{"comic-neue-bold.ttf", genFont("@expo-google-fonts/comic-neue/700Bold/ComicNeue_700Bold.ttf")},
	}
	for _, s := range steps {
		data, err := s.gen()
		if err != nil {
			return fmt.Errorf("%s: %w", s.out, err)
		}
		// 0o600 keeps gosec happy; these are build-time artifacts, embedded then discarded.
		if err := os.WriteFile(filepath.Join(assetsDir, s.out), data, 0o600); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d bytes)\n", s.out, len(data))
	}
	return nil
}

func genMarked() ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(nodeModules, "marked", "lib", "marked.umd.js"))
	if err != nil {
		return nil, err
	}
	return sourceMapRe.ReplaceAll(b, nil), nil
}

func genMarkdownCSS() ([]byte, error) {
	return os.ReadFile(filepath.Join(nodeModules, "github-markdown-css", "github-markdown.css"))
}

// genFont copies a TrueType font verbatim from node_modules. The vendored font packages
// (dejavu-fonts-ttf, @expo-google-fonts/*) ship the static .ttf files the badge (sfnt) and
// graph (freetype) renderers parse; @fontsource ships only woff2, which neither can read.
func genFont(rel string) func() ([]byte, error) {
	return func() ([]byte, error) {
		return os.ReadFile(filepath.Join(nodeModules, filepath.FromSlash(rel)))
	}
}

// genMDI builds the Material Design Icons table from @mdi/svg.
func genMDI() ([]byte, error) {
	return genIconSet("MDI", filepath.Join(nodeModules, "@mdi", "svg", "svg"))
}

// genSimpleIcons builds the Simple Icons table from simple-icons. Each file is named
// by slug and holds a single 24x24 <path> alongside a <title>, like @mdi/svg.
func genSimpleIcons() ([]byte, error) {
	return genIconSet("Simple Icons", filepath.Join(nodeModules, "simple-icons", "icons"))
}

// genIconSet reads every single-path SVG glyph under dir (filename minus ".svg" is the
// icon name) and builds a gzipped, sorted "name\tpath" table. label is used only in the
// progress line.
func genIconSet(label, dir string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]string, len(entries))
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		m := svgPathRe.FindSubmatch(data)
		if m == nil {
			return nil, fmt.Errorf("no path in %s", e.Name())
		}
		name := strings.TrimSuffix(e.Name(), ".svg")
		paths[name] = string(m[1])
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no icons in %s", dir)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	bw := bufio.NewWriter(zw)
	for _, name := range names {
		if _, err := fmt.Fprintf(bw, "%s\t%s\n", name, paths[name]); err != nil {
			return nil, err
		}
	}
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	fmt.Printf("      (%d %s icons)\n", len(names), label)
	return buf.Bytes(), nil
}
