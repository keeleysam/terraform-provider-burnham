package cel

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/ext"
)

// evalOptions controls how Eval renders its result and bounds evaluation cost.
type evalOptions struct {
	tsFormat    string // "rfc3339" (default) | "unix"
	durFormat   string // "string" (default, e.g. "5400s") | "go" ("1h30m0s") | "seconds" (number)
	bytesFormat string // "base64" (default) | "hex"
	costLimit   uint64 // 0 = cel-go default (unbounded)
}

func defaultEvalOptions() evalOptions {
	return evalOptions{tsFormat: "rfc3339", durFormat: "string", bytesFormat: "base64"}
}

// evalEnvOptions is the fixed set of cel-go standard + extension libraries Eval enables.
// It is standard CEL only: dialect-specific functions (GCP/k8s custom) are not available and such expressions fail to compile.
// Every library here is deterministic (no wall clock, no randomness); do not add a non-deterministic extension, as that would break plan stability.
func evalEnvOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.OptionalTypes(),
		ext.Strings(),
		ext.Math(),
		ext.Lists(),
		ext.Sets(),
		ext.Encoders(),
		ext.Bindings(),
		ext.TwoVarComprehensions(),
		ext.Regex(),
		ext.Network(),
	}
}

// Eval compiles and evaluates a standard-CEL expression against the given variable bindings and returns the result as a Go value (bool, int64, uint64, float64, string, []any, map[string]any).
// Variables are declared as dyn, so a referenced variable absent from vars is a compile error, as is any undefined function.
// Evaluation is deterministic (the enabled libraries have no wall clock or randomness).
func Eval(expr string, vars map[string]any, opts evalOptions) (any, error) {
	envOpts := evalEnvOptions()
	for name := range vars {
		envOpts = append(envOpts, cel.Variable(name, cel.DynType))
	}
	env, err := cel.NewEnv(envOpts...)
	if err != nil {
		return nil, err
	}

	ast, iss := env.Compile(expr)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}

	var progOpts []cel.ProgramOption
	if opts.costLimit > 0 {
		progOpts = append(progOpts, cel.CostLimit(opts.costLimit))
	}
	prg, err := env.Program(ast, progOpts...)
	if err != nil {
		return nil, err
	}

	if vars == nil {
		vars = map[string]any{}
	}
	out, _, err := prg.Eval(vars)
	if err != nil {
		return nil, err
	}
	return refValToGo(out, opts)
}

// refValToGo converts a CEL result value into a Go value the provider can return.
func refValToGo(v ref.Val, opts evalOptions) (any, error) {
	switch t := v.(type) {
	case types.Bool:
		return bool(t), nil
	case types.Int:
		return int64(t), nil
	case types.Uint:
		return uint64(t), nil
	case types.Double:
		return float64(t), nil
	case types.String:
		return string(t), nil
	case types.Null:
		return nil, nil
	case types.Bytes:
		return formatBytes([]byte(t), opts.bytesFormat)
	case types.Duration:
		return formatDuration(t.Duration, opts.durFormat), nil
	case types.Timestamp:
		return formatTimestamp(t.Time, opts.tsFormat), nil
	case *types.Optional:
		if !t.HasValue() {
			return nil, nil
		}
		return refValToGo(t.GetValue(), opts)
	}

	if lister, ok := v.(traits.Lister); ok {
		n := int64(lister.Size().(types.Int))
		out := make([]any, 0, n)
		for i := int64(0); i < n; i++ {
			el, err := refValToGo(lister.Get(types.Int(i)), opts)
			if err != nil {
				return nil, err
			}
			out = append(out, el)
		}
		return out, nil
	}

	if mapper, ok := v.(traits.Mapper); ok {
		out := make(map[string]any)
		it := mapper.Iterator()
		for it.HasNext() == types.True {
			k := it.Next()
			ks, ok := k.(types.String)
			if !ok {
				return nil, fmt.Errorf("map key %v is not a string; a Terraform object requires string keys", k.Value())
			}
			val, _ := mapper.Find(k)
			gv, err := refValToGo(val, opts)
			if err != nil {
				return nil, err
			}
			out[string(ks)] = gv
		}
		return out, nil
	}

	return nil, fmt.Errorf("unsupported CEL result type %s", v.Type().TypeName())
}

func formatBytes(b []byte, format string) (string, error) {
	switch format {
	case "hex":
		return hex.EncodeToString(b), nil
	case "base64", "":
		return base64.StdEncoding.EncodeToString(b), nil
	}
	return "", fmt.Errorf("unknown bytes_format %q; expected base64 or hex", format)
}

func formatDuration(d time.Duration, format string) any {
	switch format {
	case "go":
		return d.String()
	case "seconds":
		return d.Seconds()
	default: // "string": a seconds form like "5400s"; float-derived, not the protobuf-exact serializer, so very large nanosecond durations lose precision.
		return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) + "s"
	}
}

func formatTimestamp(tm time.Time, format string) any {
	switch format {
	case "unix":
		return tm.Unix()
	default: // "rfc3339": RFC3339Nano so sub-second precision is preserved; it omits trailing zeros, so whole-second timestamps render identically to RFC3339.
		return tm.UTC().Format(time.RFC3339Nano)
	}
}
