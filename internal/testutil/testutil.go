// Package testutil holds small cross-package test helpers. Like promtest it is a
// normal package (not a _test.go file) so build-tagged tests can import it, but it
// is referenced only from test code and never linked into the kromgo binary.
package testutil

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// FreePort returns a TCP port that was free at call time. It binds :0, reads the
// assigned port, then releases it — so the port is available for a server to bind,
// at the usual (small) risk of a race before that bind happens.
func FreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

// ModuleRoot walks up from the test's working directory to the directory holding go.mod.
func ModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod")
		}
		dir = parent
	}
}
