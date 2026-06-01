package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"nonsense", slog.LevelInfo},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, parseLevel(tc.in))
		})
	}
}

func TestNewHandler(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		format      string
		wantJSON    bool
		debugLogged bool
		level       string
	}{
		{name: "json default", format: "", wantJSON: true, level: "info", debugLogged: false},
		{name: "text", format: "text", wantJSON: false, level: "info", debugLogged: false},
		{name: "debug level emits debug", format: "json", wantJSON: true, level: "debug", debugLogged: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := slog.New(newHandler(&buf, tc.level, tc.format))
			log.Debug("dbg")
			log.Info("nfo", "k", "v")

			out := buf.String()
			if tc.wantJSON {
				assert.Contains(t, out, `"msg":"nfo"`)
				var rec map[string]any
				require.NoError(t, json.Unmarshal([]byte(firstLine(out)), &rec))
			} else {
				assert.Contains(t, out, "msg=nfo")
			}
			assert.Equal(t, tc.debugLogged, bytes.Contains([]byte(out), []byte("dbg")))
		})
	}
}

func TestFromContext_Fallback(t *testing.T) {
	t.Parallel()
	assert.Same(t, slog.Default(), FromContext(context.Background()))
}

func TestWithLogger_RoundTrip(t *testing.T) {
	t.Parallel()
	want := slog.New(newHandler(&bytes.Buffer{}, "info", "json"))
	ctx := WithLogger(context.Background(), want)
	assert.Same(t, want, FromContext(ctx))
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	return s
}
