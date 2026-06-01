package kromgo

import (
	"strings"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorNameToHex(t *testing.T) {
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
			assert.Equal(t, want, colorNameToHex(in))
		})
	}
}

func TestNewBadgeRenderer_DefaultFont(t *testing.T) {
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)
	require.NotNil(t, r)
	svg := string(r.render(config.StyleFlat, "", "label", "msg", "green"))
	assert.True(t, strings.HasPrefix(svg, "<svg"))
	assert.Contains(t, svg, ">label<")
	assert.Contains(t, svg, ">msg<")
	assert.Contains(t, svg, "textLength=", "text width pinned to prevent overflow")
	assert.Contains(t, svg, "viewBox=", "scalable")
}

func TestNewBadgeRenderer_UnknownFont(t *testing.T) {
	_, err := newBadgeRenderer(config.BadgeDefaults{Font: "not-a-font"})
	assert.Error(t, err)
}

func TestNewBadgeRenderer_NamedFont(t *testing.T) {
	r, err := newBadgeRenderer(config.BadgeDefaults{Font: "go-bold"})
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestBadgeRender_Icon(t *testing.T) {
	r, err := newBadgeRenderer(config.BadgeDefaults{})
	require.NoError(t, err)

	path, err := resolveIcon("mdi:server-outline")
	require.NoError(t, err)
	svg := string(r.render(config.StyleFlat, path, "", "online", "green"))

	assert.Contains(t, svg, path, "icon path should be embedded")
	assert.Contains(t, svg, `fill="#fff"`)
	assert.Contains(t, svg, ">online<")
}

func TestResolveIcon(t *testing.T) {
	path, err := resolveIcon("mdi:server-outline")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, "M"))

	empty, err := resolveIcon("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	_, err = resolveIcon("mdi:does-not-exist")
	assert.Error(t, err)

	_, err = resolveIcon("server-outline") // missing mdi: prefix
	assert.Error(t, err)
}
