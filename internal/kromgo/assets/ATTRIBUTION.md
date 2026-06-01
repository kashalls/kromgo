# Embedded assets

These third-party files are **fetched at build time** by `cmd/genassets`
(`mise run assets`) and embedded into the binary with `//go:embed` — they are **not
committed** to the repo (see the root `.gitignore`). A fresh checkout, every CI run,
and the Docker build all pull them before compiling; locally, `mise run assets`
fetches any that are missing (`-force` re-fetches after a version bump). The built
binary is self-contained — nothing is fetched at runtime.

The gallery serves `marked.min.js` + `github-markdown.css` (and the first-party
`gallery.css` / `gallery.js`) from `/assets/`. `mdi.txt.gz` is the icon data —
embedded but **not** web-served, decoded into a name→path map on first use.

| File                  | Upstream                                                                   | Version | License    | Source                                                                   |
| --------------------- | -------------------------------------------------------------------------- | ------- | ---------- | ------------------------------------------------------------------------ |
| `marked.min.js`       | [marked](https://github.com/markedjs/marked)                               | 18.0.4  | MIT        | jsDelivr `marked@<v>/lib/marked.umd.min.js` (sourcemap comment stripped) |
| `github-markdown.css` | [github-markdown-css](https://github.com/sindresorhus/github-markdown-css) | 5.9.0   | MIT        | jsDelivr `github-markdown-css@<v>/github-markdown.css`                   |
| `mdi.txt.gz`          | [@mdi/svg](https://github.com/Templarian/MaterialDesign-SVG)               | 7.4.47  | Apache-2.0 | npm `@mdi/svg` tarball → gzipped `name⇥path` table (one glyph/line)      |

Pinned versions live as constants in `cmd/genassets`; bump them there and run
`mise run assets -force` (or rely on the next clean CI/Docker build).

`gallery.css` and `gallery.js` are first-party — they live in this repository.
