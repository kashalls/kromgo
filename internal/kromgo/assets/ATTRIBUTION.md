# Vendored gallery assets

These files are embedded into the binary (`//go:embed`) and served from `/assets/`
for the index gallery page. They are vendored (not fetched at runtime) so the
service stays self-contained, works air-gapped, and keeps a strict
`script-src 'self'` Content-Security-Policy. To update, replace the file with the
same pinned-version URL below and adjust the version here.

| File                  | Upstream                                                                   | Version | License | Source                                                                       |
| --------------------- | -------------------------------------------------------------------------- | ------- | ------- | ---------------------------------------------------------------------------- |
| `marked.min.js`       | [marked](https://github.com/markedjs/marked)                               | 18.0.4  | MIT     | `https://cdn.jsdelivr.net/npm/marked@18.0.4/lib/marked.umd.min.js`           |
| `github-markdown.css` | [github-markdown-css](https://github.com/sindresorhus/github-markdown-css) | 5.9.0   | MIT     | `https://cdn.jsdelivr.net/npm/github-markdown-css@5.9.0/github-markdown.css` |

The jsDelivr `sourceMappingURL` comment is stripped from `marked.min.js` so the
file references nothing external.

`gallery.css` and `gallery.js` are first-party (part of this repository).
