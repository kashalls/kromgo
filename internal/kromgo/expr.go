package kromgo

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
)

// newCELEnv builds the CEL environment exposed to a metric's value/color
// expressions. Expressions get two variables — result (the sample value) and
// labels (its label set) — plus the standard string extension and kromgo's
// humanizer functions. CEL is sandboxed: no env/file/network access.
func newCELEnv() (*cel.Env, error) {
	return cel.NewEnv(append([]cel.EnvOption{
		cel.Variable("result", cel.DoubleType),
		cel.Variable("labels", cel.MapType(cel.StringType, cel.StringType)),
		ext.Strings(),
	}, humanizerFuncs()...)...)
}

// humanizerFuncs registers kromgo's formatting helpers as CEL functions that take
// the numeric result and return a string, e.g. humanizeBytes(result).
func humanizerFuncs() []cel.EnvOption {
	fn := func(name string, impl func(float64) string) cel.EnvOption {
		return cel.Function(name, cel.Overload(name+"_double",
			[]*cel.Type{cel.DoubleType}, cel.StringType,
			cel.UnaryBinding(func(v ref.Val) ref.Val {
				return types.String(impl(float64(v.(types.Double))))
			})))
	}
	return []cel.EnvOption{
		fn("humanizeBytes", humanizeBytes),
		fn("humanizeSIBytes", humanizeSIBytes),
		fn("humanizeNumber", humanizeNumber),
		fn("humanizeFloat", humanizeFloat),
		fn("humanizeDuration", humanizeDuration),
	}
}

// compileStringExpr compiles src and requires it to evaluate to a string. kind/name
// are used only for error context.
func compileStringExpr(env *cel.Env, name, kind, src string) (cel.Program, error) {
	ast, iss := env.Compile(src)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("metric %q %s expression: %w", name, kind, iss.Err())
	}
	if ast.OutputType() != cel.StringType {
		return nil, fmt.Errorf("metric %q %s expression must return string, got %s", name, kind, ast.OutputType())
	}
	prog, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("metric %q %s expression: %w", name, kind, err)
	}
	return prog, nil
}

// evalStringExpr evaluates prog against the sample value and labels.
func evalStringExpr(prog cel.Program, result float64, labels map[string]string) (string, error) {
	out, _, err := prog.Eval(map[string]any{"result": result, "labels": labels})
	if err != nil {
		return "", err
	}
	s, ok := out.Value().(string)
	if !ok {
		return "", fmt.Errorf("expression returned %T, want string", out.Value())
	}
	return s, nil
}
