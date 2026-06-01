# Embedded assets

These third-party assets are embedded into the binary with `//go:embed` but are **not
committed**. They are vendored via npm (`package.json` + `package-lock.json`, kept
current by Renovate) and built by `cmd/genassets`, which reads `node_modules` and does
no network I/O of its own. A fresh checkout, every CI run, and the Docker build run
`npm ci` then `go run ./cmd/genassets` before compiling; locally, `mise run assets`
does both. The built binary is self-contained — nothing is fetched at runtime.

The gallery serves `marked.js` + `github-markdown.css` (and the first-party
`gallery.css` / `gallery.js`) from `/assets/`. `mdi.txt.gz` is the icon data —
embedded but **not** web-served, decoded into a name→path map on first use.

| File                  | npm package                                                                | License    | Built from                                                               |
| --------------------- | -------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------ |
| `marked.js`           | [marked](https://github.com/markedjs/marked)                               | MIT        | `node_modules/marked/lib/marked.umd.js` (sourcemap comment stripped)     |
| `github-markdown.css` | [github-markdown-css](https://github.com/sindresorhus/github-markdown-css) | MIT        | `node_modules/github-markdown-css/github-markdown.css`                   |
| `mdi.txt.gz`          | [@mdi/svg](https://github.com/Templarian/MaterialDesign-SVG)               | Apache-2.0 | `node_modules/@mdi/svg/svg/*.svg` → gzipped `name⇥path` table (one/line) |

Versions are pinned in `package.json` / `package-lock.json`; Renovate opens PRs to bump
them. `gallery.css` and `gallery.js` are first-party — they live in this repository.
