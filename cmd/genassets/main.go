// Command genassets fetches the third-party assets that are embedded into the
// binary but intentionally NOT committed to the repo: the gallery's marked.js +
// github-markdown.css, and the full Material Design Icons set (as a gzipped
// name\tpath table). Because internal/kromgo embeds these with //go:embed, this
// must run before any `go build`/`go test`/`go vet` of that package.
//
//	go run ./cmd/genassets         # fetch any assets not already present
//	go run ./cmd/genassets -force  # re-fetch all (e.g. after bumping a version)
//
// A fresh checkout (CI, Docker, clone) has none of these files, so they are always
// pulled at build time; -force refreshes them locally after a version bump.
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Pinned upstream versions. Bump here, then `go run ./cmd/genassets -force`.
const (
	markedVersion  = "18.0.4"
	markdownCSSVer = "5.9.0"
	mdiVersion     = "7.4.47"

	jsdelivr  = "https://cdn.jsdelivr.net/npm/"
	assetsDir = "internal/kromgo/assets"
)

var (
	force       = flag.Bool("force", false, "re-download assets even if already present")
	sourceMapRe = regexp.MustCompile(`(?m)^\s*//[#@]\s*sourceMappingURL=.*$\n?`)
	mdiPathRe   = regexp.MustCompile(`\bd="([^"]+)"`)
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genassets:", err)
		os.Exit(1)
	}
}

func run() error {
	assets := []struct {
		name string
		gen  func() ([]byte, error)
	}{
		{"marked.min.js", fetchMarked},
		{"github-markdown.css", fetchMarkdownCSS},
		{"mdi.txt.gz", fetchMDI},
	}
	for _, a := range assets {
		if err := write(a.name, a.gen); err != nil {
			return err
		}
	}
	return nil
}

// write generates assetsDir/name, unless it already exists and -force is unset.
func write(name string, gen func() ([]byte, error)) (err error) {
	path := filepath.Join(assetsDir, name)
	if !*force {
		if _, statErr := os.Stat(path); statErr == nil {
			fmt.Printf("skip  %s (present; -force to refresh)\n", name)
			return nil
		}
	}
	data, err := gen()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	fmt.Printf("write %s (%d bytes)\n", name, len(data))
	return nil
}

// readBody handles an http.Get result: it takes (resp, err) directly so the URL
// passed to http.Get stays a constant expression (no tainted-URL lint warning).
func readBody(resp *http.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Request.URL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func fetchMarked() ([]byte, error) {
	b, err := readBody(http.Get(jsdelivr + "marked@" + markedVersion + "/lib/marked.umd.min.js"))
	if err != nil {
		return nil, err
	}
	return sourceMapRe.ReplaceAll(b, nil), nil // strip the jsDelivr sourcemap comment
}

func fetchMarkdownCSS() ([]byte, error) {
	return readBody(http.Get(jsdelivr + "github-markdown-css@" + markdownCSSVer + "/github-markdown.css"))
}

// fetchMDI downloads the pinned @mdi/svg tarball and builds a gzipped, sorted
// "name\tpath" table — one 24x24 glyph path per icon.
func fetchMDI() ([]byte, error) {
	tgz, err := readBody(http.Get("https://registry.npmjs.org/@mdi/svg/-/svg-" + mdiVersion + ".tgz"))
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	icons := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		name, ok := strings.CutPrefix(hdr.Name, "package/svg/")
		if !ok || !strings.HasSuffix(name, ".svg") {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		m := mdiPathRe.FindSubmatch(data)
		if m == nil {
			return nil, fmt.Errorf("no path in %s", hdr.Name)
		}
		icons[strings.TrimSuffix(name, ".svg")] = string(m[1])
	}
	if len(icons) == 0 {
		return nil, fmt.Errorf("no icons found in tarball (layout changed?)")
	}

	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	bw := bufio.NewWriter(zw)
	for _, name := range slices.Sorted(maps.Keys(icons)) {
		if _, err := fmt.Fprintf(bw, "%s\t%s\n", name, icons[name]); err != nil {
			return nil, err
		}
	}
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	fmt.Printf("      (%d MDI icons, %s)\n", len(icons), mdiVersion)
	return buf.Bytes(), nil
}
