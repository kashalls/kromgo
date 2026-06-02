package kromgo

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
)

// newCELEnv builds the CEL environment exposed to a metric's value/color
// expressions. Expressions get two variables — result (the sample value) and
// labels (its label set) — plus the string and math extensions, optional types,
// and kromgo's humanizer functions. CEL is sandboxed: no env/file/network access.
func newCELEnv() (*cel.Env, error) {
	return cel.NewEnv(append([]cel.EnvOption{
		cel.Variable("result", cel.DoubleType),
		cel.Variable("labels", cel.MapType(cel.StringType, cel.StringType)),
		// result is a double; allow comparing it against plain int literals
		// (result < 35, not result < 35.0) — the usual color-threshold case.
		cel.CrossTypeNumericComparisons(true),
		// Optional map indexing: labels[?"k"].orValue("n/a") for absent labels.
		cel.OptionalTypes(),
		ext.Strings(),
		// math.round/abs/least/greatest and the isNaN/isInf/isFinite guards —
		// Prometheus can return non-finite values that would otherwise render
		// literally (e.g. "NaN") on a badge.
		ext.Math(),
	}, append(humanizerFuncs(), colorFuncs()...)...)...)
}

// unaryStringFunc registers a CEL function that takes the numeric result and returns
// a string, e.g. humanizeBytes(result).
func unaryStringFunc(name string, impl func(float64) string) cel.EnvOption {
	return cel.Function(name, cel.Overload(name+"_double",
		[]*cel.Type{cel.DoubleType}, cel.StringType,
		cel.UnaryBinding(func(v ref.Val) ref.Val {
			return types.String(impl(float64(v.(types.Double))))
		})))
}

// humanizerFuncs registers kromgo's value-formatting helpers (see formatting.go).
func humanizerFuncs() []cel.EnvOption {
	return []cel.EnvOption{
		unaryStringFunc("humanizeBytes", humanizeBytes),
		unaryStringFunc("humanizeCommas", humanizeCommas),
		unaryStringFunc("humanizeFloat", humanizeFloat),
		unaryStringFunc("humanizeDuration", humanizeDuration),
		unaryStringFunc("humanizeDurationDays", humanizeDurationDays),
	}
}

// colorFuncs registers kromgo's color helper (see colors.go) for a colorExpr,
// e.g. colorExpr: colorScale(result, [35.0, 75.0], ["green", "orange", "red"]).
func colorFuncs() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function("colorScale", cel.Overload("colorScale_double_list_list",
			[]*cel.Type{cel.DoubleType, cel.ListType(cel.DoubleType), cel.ListType(cel.StringType)}, cel.StringType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				steps, err := args[1].ConvertToNative(reflect.TypeOf([]float64(nil)))
				if err != nil {
					return types.NewErr("colorScale steps: %v", err)
				}
				colors, err := args[2].ConvertToNative(reflect.TypeOf([]string(nil)))
				if err != nil {
					return types.NewErr("colorScale colors: %v", err)
				}
				return types.String(colorScale(float64(args[0].(types.Double)), steps.([]float64), colors.([]string)))
			}))),
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
