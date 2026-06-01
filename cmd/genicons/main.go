// Command genicons regenerates the embedded Material Design Icons data file
// (internal/kromgo/assets/mdi.txt.gz) from the pinned @mdi/svg npm tarball. The
// output is a gzipped, tab-separated "name\tpath" table — one line per icon —
// loaded into a map at startup by the kromgo package.
//
// Run from the repository root:
//
//	go run ./cmd/genicons
//
// Bump mdiVersion (and assets/ATTRIBUTION.md) to update the icon set.
package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
)

const (
	mdiVersion = "7.4.47"
	tarballURL = "https://registry.npmjs.org/@mdi/svg/-/svg-" + mdiVersion + ".tgz"
	outPath    = "internal/kromgo/assets/mdi.txt.gz"
)

// pathRe extracts the geometry from an MDI glyph's single <path d="…">. The \b
// avoids matching the "d" in attributes like id/width.
var pathRe = regexp.MustCompile(`\bd="([^"]+)"`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genicons:", err)
		os.Exit(1)
	}
}

func run() (err error) {
	resp, err := http.Get(tarballURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", tarballURL, resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)

	icons := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name, ok := strings.CutPrefix(hdr.Name, "package/svg/")
		if !ok || !strings.HasSuffix(name, ".svg") {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		m := pathRe.FindSubmatch(data)
		if m == nil {
			return fmt.Errorf("no path in %s", hdr.Name)
		}
		icons[strings.TrimSuffix(name, ".svg")] = string(m[1])
	}
	if len(icons) == 0 {
		return fmt.Errorf("no icons found in tarball (layout changed?)")
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr // surface a final-flush error on the success path
		}
	}()

	zw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(zw)
	for _, name := range slices.Sorted(maps.Keys(icons)) {
		if _, err := fmt.Fprintf(bw, "%s\t%s\n", name, icons[name]); err != nil {
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	fmt.Printf("wrote %d icons (MDI %s) to %s\n", len(icons), mdiVersion, outPath)
	return nil
}
