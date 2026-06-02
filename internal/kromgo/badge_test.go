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
	svg := string(r.render(config.StyleFlat, "", "label", "msg", "green"))
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
	svg := string(r.render(config.StyleFlat, path, "", "online", "green"))

	assert.Contains(t, svg, path, "icon path should be embedded")
	assert.Contains(t, svg, `fill="#fff"`)
}

func TestBadgeRender_Accessibility(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	svg := string(r.render(config.StyleFlat, "", "build", "passing", "green"))
	assert.Contains(t, svg, `role="img"`)
	assert.Contains(t, svg, `aria-label="build: passing"`, "screen readers get a combined label")
	assert.Contains(t, svg, `<title>build: passing</title>`, "native hover tooltip")

	// A message-only badge labels with just the message.
	msgOnly := string(r.render(config.StyleFlat, "", "", "online", "green"))
	assert.Contains(t, msgOnly, `aria-label="online"`)
}

func TestBadgeRender_TextContrast(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	// Dark background (default blue) → white text, near-black shadow.
	dark := string(r.render(config.StyleFlat, "", "label", "msg", "blue"))
	assert.Contains(t, dark, `<path fill="#fff"`, "white text on a dark badge")
	assert.NotContains(t, dark, `<path fill="#333"`)

	// Light custom background → dark text + light shadow (matches shields.io).
	light := string(r.render(config.StyleFlat, "", "label", "msg", "#eeeeee"))
	assert.Contains(t, light, `<path fill="#333"`, "dark text on a light badge")
	assert.Contains(t, light, `fill="#ccc" fill-opacity=".3"`, "light shadow on a light badge")
	// The label segment stays dark (#555), so its text is still white.
	assert.Contains(t, light, `<path fill="#fff"`, "label text stays white on the dark label segment")
}

func TestBadgeRender_WellFormedXML(t *testing.T) {
	t.Parallel()
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	// A label full of XML metacharacters must not break the document: it flows into the
	// escaped aria-label/<title>, so the whole SVG has to stay well-formed.
	svg := r.render(config.StyleFlat, "", `a "quote" & <tag>`, "ok", "#eee")
	dec := xml.NewDecoder(bytes.NewReader(svg))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err, "rendered SVG must be well-formed XML")
	}
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
