package kromgo

import (
	"log/slog"
	"os"
	"testing"
)

// TestMain silences the intentional error-path logs these tests trigger so test
// output stays readable.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	os.Exit(m.Run())
}
