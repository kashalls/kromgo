package kromgo

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorNameToHex(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":          "#007ec6", // default blue
		"blue":      "#007ec6",
		"green":     "#97ca00",
		"red":       "#e05d44",
		"inactive":  "#9f9f9f",
		"#a1b2c3":   "#a1b2c3", // valid hex passes through
		"#zzz":      "#97ca00", // invalid hex → fallback green
		"notacolor": "#97ca00", // unknown name → fallback green
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, want, colorNameToHex(in))
		})
	}
}

func TestNewBadgeRenderer_DefaultFont(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)
	require.NotNil(t, r)
	svg := string(r.render(badgeSpec{style: config.StyleFlat, label: "label", message: "msg", color: "green"}))
	assert.True(t, strings.HasPrefix(svg, "<svg"))
	assert.Contains(t, svg, "viewBox=", "scalable")
	assert.Contains(t, svg, `<path fill="#fff"`, "text rendered as glyph paths")
	assert.NotContains(t, svg, "<text", "text is paths, not <text> elements")
}

func TestNewBadgeRenderer_UnknownFont(t *testing.T) {
	t.Parallel()
	_, err := newBadgeRenderer(config.BadgeDefaults{Font: "not-a-font"})
	assert.Error(t, err)
}

func TestNewBadgeRenderer_NamedFont(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{Font: "dejavu-sans-bold"})
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestBadgeRender_Icon(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	path, err := resolveIcon("mdi:server-outline")
	require.NoError(t, err)
	svg := string(r.render(badgeSpec{style: config.StyleFlat, iconPath: path, message: "online", color: "green"}))

	assert.Contains(t, svg, path, "icon path should be embedded")
	assert.Contains(t, svg, `fill="#fff"`)
}

func TestBadgeRender_IconNoLabel(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)
	icon, err := resolveIcon("mdi:server-outline")
	require.NoError(t, err)

	// An icon with no label is a single-color badge: the icon rides on the message
	// segment, with no separate grey label box. With a light message color the icon
	// is drawn dark — proving it's colored for the message background, not the grey
	// label background (which would render it white).
	light := string(r.render(badgeSpec{
		style: config.StyleFlat, iconPath: icon, message: "online", color: "#eeeeee", id: "x",
	}))
	assert.Contains(t, light, `<path fill="#333" d="`+icon+`"`, "icon takes the light message background's dark fill")
	assert.NotContains(t, light, `fill="#555"`, "no default grey label segment when there's no label")
	assert.Contains(t, light, `<rect x="0" `, "the single message segment spans from the badge's left edge")
}

func TestBadgeRender_Accessibility(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	svg := string(r.render(badgeSpec{style: config.StyleFlat, label: "build", message: "passing", color: "green"}))
	assert.Contains(t, svg, `role="img"`)
	assert.Contains(t, svg, `aria-label="build: passing"`, "screen readers get a combined label")
	assert.Contains(t, svg, `<title>build: passing</title>`, "native hover tooltip")

	// A message-only badge labels with just the message.
	msgOnly := string(r.render(badgeSpec{style: config.StyleFlat, message: "online", color: "green"}))
	assert.Contains(t, msgOnly, `aria-label="online"`)
}

func TestBadgeRender_TextContrast(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	// Dark background (default blue) → white text, near-black shadow.
	dark := string(r.render(badgeSpec{style: config.StyleFlat, label: "label", message: "msg", color: "blue"}))
	assert.Contains(t, dark, `<path fill="#fff"`, "white text on a dark badge")
	assert.NotContains(t, dark, `<path fill="#333"`)

	// Light custom background → dark text + light shadow (matches shields.io).
	light := string(r.render(badgeSpec{style: config.StyleFlat, label: "label", message: "msg", color: "#eeeeee"}))
	assert.Contains(t, light, `<path fill="#333"`, "dark text on a light badge")
	assert.Contains(t, light, `fill="#ccc" fill-opacity=".3"`, "light shadow on a light badge")
	// The label segment stays dark (#555), so its text is still white.
	assert.Contains(t, light, `<path fill="#fff"`, "label text stays white on the dark label segment")
}

func TestBadgeRender_LabelColor(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)
	icon, err := resolveIcon("mdi:server-outline")
	require.NoError(t, err)

	// labelColor is the resolved hex for the left segment (resolveBadge resolves names).
	// A light labelColor paints the left rect and flips its text — and the icon — dark,
	// while the message side stays independent.
	light := string(r.render(badgeSpec{
		style: config.StyleFlat, iconPath: icon, label: "build", message: "passing",
		color: "blue", labelColor: "#e0e0e0", id: "x",
	}))
	assert.Contains(t, light, `fill="#e0e0e0"`, "label segment uses labelColor")
	assert.Contains(t, light, icon, "icon embedded")
	assert.Contains(t, light, `<path fill="#333"`, "label text + icon go dark on a light label")
	assert.Contains(t, light, `<path fill="#fff"`, "message text stays white on dark blue")

	// Empty labelColor falls back to the default grey segment (all text white).
	def := string(r.render(badgeSpec{style: config.StyleFlat, label: "build", message: "passing", color: "blue", id: "x"}))
	assert.Contains(t, def, `fill="#555"`, "default label segment is grey")
	assert.NotContains(t, def, `<path fill="#333"`, "no dark text on an all-dark badge")
}

func TestBadgeRender_UniqueIDs(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	// Element ids are namespaced by badge id so two badges inlined in one document
	// don't have the second's url(#…) refs resolve to the first's defs.
	svg := string(r.render(badgeSpec{style: config.StyleFlat, label: "l", message: "m", color: "blue", id: "cpu"}))
	assert.Contains(t, svg, `id="g-cpu"`)
	assert.Contains(t, svg, `fill="url(#g-cpu)"`)
	assert.Contains(t, svg, `id="r-cpu"`)
	assert.Contains(t, svg, `clip-path="url(#r-cpu)"`)

	// A '.' (valid in a kromgo id) is sanitized to '-' for the SVG element id.
	dotted := string(r.render(badgeSpec{style: config.StyleFlat, label: "l", message: "m", color: "blue", id: "a.b"}))
	assert.Contains(t, dotted, `id="r-a-b"`)
	assert.NotContains(t, dotted, `#r-a.b`)
}

func TestBadgeRender_WellFormedXML(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	// A label full of XML metacharacters must not break the document: it flows into the
	// escaped aria-label/<title>, so the whole SVG has to stay well-formed.
	svg := r.render(badgeSpec{style: config.StyleFlat, label: `a "quote" & <tag>`, message: "ok", color: "#eee"})
	dec := xml.NewDecoder(bytes.NewReader(svg))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err, "rendered SVG must be well-formed XML")
	}
}

func TestBadgeRenderError(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	// 4xx → red ("your request is wrong"); 5xx → grey ("couldn't get an answer").
	client := string(r.renderError("cpu", "Not Found", 404))
	assert.Contains(t, client, "#e05d44", "client errors are red")
	assert.Contains(t, client, `aria-label="cpu: Not Found"`)

	server := string(r.renderError("cpu", "Query Error", 500))
	assert.Contains(t, server, "#9f9f9f", "server/upstream errors are grey")
	assert.Contains(t, server, `aria-label="cpu: Query Error"`)
}

func TestResolveIcon(t *testing.T) {
	t.Parallel()
	path, err := resolveIcon("mdi:server-outline")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, "M"))

	// Any icon from the full embedded MDI set resolves, not just a curated few.
	rocket, err := resolveIcon("mdi:rocket-launch")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(rocket, "M"))

	// Simple Icons brand logos resolve from the si: set.
	si, err := resolveIcon("si:kubernetes")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(si, "M"))

	empty, err := resolveIcon("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	_, err = resolveIcon("mdi:does-not-exist")
	assert.Error(t, err)

	_, err = resolveIcon("si:does-not-exist")
	assert.Error(t, err)

	_, err = resolveIcon("nope:server-outline") // unknown icon set
	assert.Error(t, err)

	_, err = resolveIcon("server-outline") // missing set prefix
	assert.Error(t, err)
}

func TestIconSetsEmbedded(t *testing.T) {
	t.Parallel()
	// Both full sets are embedded and decode cleanly.
	assert.Greater(t, len(mdiIcons()), 7000, "full MDI set should be embedded")
	assert.Greater(t, len(siIcons()), 3000, "full Simple Icons set should be embedded")
}
