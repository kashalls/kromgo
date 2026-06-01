package kromgo

import (
	"testing"

	"github.com/essentialkaos/go-badge"
	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorNameToHex(t *testing.T) {
	cases := map[string]string{
		"":          badge.COLOR_BLUE, // default
		"blue":      badge.COLOR_BLUE,
		"green":     badge.COLOR_GREEN,
		"red":       badge.COLOR_RED,
		"inactive":  badge.COLOR_INACTIVE,
		"#a1b2c3":   "#a1b2c3",         // valid hex passes through
		"#zzz":      badge.COLOR_GREEN, // invalid hex → fallback
		"notacolor": badge.COLOR_GREEN, // unknown name → fallback
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, colorNameToHex(in))
		})
	}
}

func TestNewBadgePool_DefaultFont(t *testing.T) {
	// No font configured → embedded default font is used.
	pool, err := newBadgePool(config.BadgeDefaults{})
	require.NoError(t, err)
	require.NotNil(t, pool)

	gen := pool.pool.Get().(*badge.Generator)
	assert.NotEmpty(t, gen.GenerateFlat("label", "msg", badge.COLOR_GREEN))
}

func TestNewBadgePool_MissingFontFile(t *testing.T) {
	_, err := newBadgePool(config.BadgeDefaults{Font: "/no/such/font.ttf"})
	assert.Error(t, err)
}
