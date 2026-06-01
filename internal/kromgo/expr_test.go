package kromgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eval(t *testing.T, src string, result float64, labels map[string]string) (string, error) {
	t.Helper()
	env, err := newCELEnv()
	require.NoError(t, err)
	prog, err := compileStringExpr(env, "test", "value", src)
	require.NoError(t, err)
	return evalStringExpr(prog, result, labels)
}

func TestCEL_Expressions(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		result float64
		labels map[string]string
		want   string
	}{
		{"suffix", `string(result) + "%"`, 17.5, nil, "17.5%"},
		{"ternary color", `result <= 50.0 ? "green" : "red"`, 17.5, nil, "green"},
		{"ternary high", `result <= 50.0 ? "green" : "red"`, 80, nil, "red"},
		{"label", `labels["version"]`, 0, map[string]string{"version": "v1.2.3"}, "v1.2.3"},
		{"humanBytes", `humanBytes(result)`, 1572864, nil, "1.5 MiB"},
		{"humanDuration", `humanDuration(result)`, 9000, nil, "2h30m"},
		{"humanizeAge", `humanizeAge(result)`, 467 * 86400, nil, "1y3m12d"},
		{"humanizeThousands", `humanizeThousands(result)`, 1000000, nil, "1,000,000"},
		{"humanizeFtoa", `humanizeFtoa(result)`, 2.5, nil, "2.5"},
		{"string method", `labels["x"].startsWith("v") ? "yes" : "no"`, 0, map[string]string{"x": "v1"}, "yes"},
		{"safe label", `"version" in labels ? labels["version"] : "unknown"`, 0, nil, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := eval(t, tc.src, tc.result, tc.labels)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCEL_CompileRejectsNonString(t *testing.T) {
	env, err := newCELEnv()
	require.NoError(t, err)
	_, err = compileStringExpr(env, "test", "value", "result") // double, not string
	assert.Error(t, err)
}

func TestCEL_NoEnvOrFileAccess(t *testing.T) {
	// CEL is sandboxed: there is no env()/readFile()/etc. to leak host state.
	env, err := newCELEnv()
	require.NoError(t, err)
	for _, src := range []string{`env("HOME")`, `readFile("/etc/passwd")`, `getHostByName("x")`} {
		_, err := compileStringExpr(env, "test", "value", src)
		assert.Error(t, err, "expected %q to be undefined in the CEL env", src)
	}
}
