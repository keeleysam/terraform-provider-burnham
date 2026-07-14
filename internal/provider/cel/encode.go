package cel

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"

	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"
	"github.com/google/cel-go/parser"
)

// newParseEnv builds the CEL environment used to validate and canonicalize output.
// It enables optional types (so `?.`, `[?]`, `[?x]`, `{?k: v}`, optional.*) and two-variable comprehensions (`m.all(k, v, ...)`, transformList/Map/MapEntry) so real Kubernetes and GCP dialects validate.
// Validation stays syntax-only: unknown functions/variables are never rejected.
func newParseEnv(trackMacros bool) (*celgo.Env, error) {
	opts := []celgo.EnvOption{celgo.OptionalTypes(), ext.TwoVarComprehensions()}
	if trackMacros {
		opts = append(opts, celgo.EnableMacroCallTracking())
	}
	return celgo.NewEnv(opts...)
}

// encoder walks a decoded HCL data tree and builds a cel-go AST.
type encoder struct {
	f      ast.ExprFactory
	nextID int64
}

func (e *encoder) id() int64 { e.nextID++; return e.nextID }

// encode dispatches on the Go type of a node.
// Bare scalars and lists are literals; a single-key object is a node keyed by its kind.
func (e *encoder) encode(node any) (ast.Expr, error) {
	switch v := node.(type) {
	case nil:
		return e.f.NewLiteral(e.id(), types.NullValue), nil
	case bool:
		return e.f.NewLiteral(e.id(), types.Bool(v)), nil
	case string:
		return e.f.NewLiteral(e.id(), types.String(v)), nil
	case int:
		return e.f.NewLiteral(e.id(), types.Int(int64(v))), nil
	case int64:
		return e.f.NewLiteral(e.id(), types.Int(v)), nil
	case float64:
		return e.numberLiteral(new(big.Float).SetFloat64(v))
	case json.Number:
		bf, _, err := big.ParseFloat(v.String(), 10, 512, big.ToNearestEven)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", v.String())
		}
		return e.numberLiteral(bf)
	case []any:
		return e.encodeList(v)
	case map[string]any:
		return e.encodeObject(v)
	default:
		return nil, fmt.Errorf("unsupported value of type %T", node)
	}
}

// numberLiteral emits an int64 literal for integral values, a double otherwise.
// Out-of-range values error rather than silently clamping (which would produce valid-but-wrong CEL).
func (e *encoder) numberLiteral(bf *big.Float) (ast.Expr, error) {
	if bf.IsInt() {
		i, acc := bf.Int64()
		if acc != big.Exact {
			return nil, fmt.Errorf("integer %s is out of range for a CEL int (64-bit signed); for large unsigned values use { const = { uint64_value = ... } }", bf.Text('f', -1))
		}
		return e.f.NewLiteral(e.id(), types.Int(i)), nil
	}
	f, _ := bf.Float64()
	if math.IsInf(f, 0) {
		return nil, fmt.Errorf("number %s is out of range for a CEL double", bf.Text('g', -1))
	}
	return e.f.NewLiteral(e.id(), types.Double(f)), nil
}

func (e *encoder) encodeList(items []any) (ast.Expr, error) {
	elems := make([]ast.Expr, len(items))
	var optIndices []int32
	for i, item := range items {
		// { optional = <expr> } marks an optional list element (CEL `[?x]`).
		if obj, ok := item.(map[string]any); ok && len(obj) == 1 {
			if inner, isOpt := obj["optional"]; isOpt {
				optIndices = append(optIndices, int32(i))
				item = inner
			}
		}
		el, err := e.encode(item)
		if err != nil {
			return nil, fmt.Errorf("list element %d: %w", i, err)
		}
		elems[i] = el
	}
	return e.f.NewList(e.id(), elems, optIndices), nil
}

func (e *encoder) encodeObject(obj map[string]any) (ast.Expr, error) {
	if len(obj) != 1 {
		return nil, fmt.Errorf("node object must have exactly one key, got %d", len(obj))
	}
	var key string
	var val any
	for k, v := range obj {
		key, val = k, v
	}
	switch key {
	case "ident":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("ident must be a string path")
		}
		return e.parseReference(s)
	case "call":
		return e.encodeCall(val)
	case "const":
		return e.literalize(val)
	case "const_expr":
		obj, ok := val.(map[string]any)
		if !ok || len(obj) != 1 {
			return nil, fmt.Errorf("const_expr must be an object with a single constant kind")
		}
		for k, v := range obj {
			if !isConstantKind(k) {
				return nil, fmt.Errorf("const_expr key %q is not a CEL constant kind", k)
			}
			return e.typedConst(k, v)
		}
		return nil, fmt.Errorf("const_expr must have a constant kind")
	case "struct":
		return e.encodeStruct(val)
	case "select":
		// Surface type-name alias for select_expr (field access on an expression operand).
		return e.encodeSelectExpr(val)
	case "list":
		// Surface type-name alias for list_expr (a bare list is the usual surface form).
		return e.encodeListExpr(val)
	case "ident_expr":
		return e.encodeIdentExpr(val)
	case "select_expr":
		return e.encodeSelectExpr(val)
	case "call_expr":
		return e.encodeCall(val)
	case "list_expr":
		return e.encodeListExpr(val)
	case "struct_expr":
		return e.encodeStructExpr(val)
	case "comprehension_expr":
		return nil, fmt.Errorf("comprehension_expr cannot be rendered directly; author comprehensions via the macro call form, e.g. { call = { target = ..., function = \"exists\", args = [ { ident = \"x\" }, <predicate> ] } }")
	case "raw":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("raw must be a CEL expression string")
		}
		return e.parseCEL(s)
	case "optional":
		return nil, fmt.Errorf("optional is only valid as a list element, e.g. [ a, { optional = b } ]")
	default:
		if isOperator(key) {
			return e.buildOperator(key, val)
		}
		return nil, fmt.Errorf("unknown node key %q", key)
	}
}

// encodeCall builds a global or receiver call.
// Macros (has/all/exists/…) are just calls whose function is a macro name, with the bound variable passed as an ident argument; cel-go unparses them to the macro sugar.
// Unknown functions need no special handling.
func (e *encoder) encodeCall(val any) (ast.Expr, error) {
	spec, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("call must be an object with function/target/args")
	}
	for k := range spec {
		if k != "function" && k != "target" && k != "args" {
			return nil, fmt.Errorf("unknown call key %q; expected function, target, or args", k)
		}
	}
	fn, ok := spec["function"].(string)
	if !ok {
		return nil, fmt.Errorf("call.function must be a string")
	}

	var args []ast.Expr
	if rawArgs, present := spec["args"]; present {
		list, ok := rawArgs.([]any)
		if !ok {
			return nil, fmt.Errorf("call.args must be a list")
		}
		args = make([]ast.Expr, len(list))
		for i, a := range list {
			built, err := e.encode(a)
			if err != nil {
				return nil, fmt.Errorf("call %q arg %d: %w", fn, i, err)
			}
			args[i] = built
		}
	}

	if tgt, present := spec["target"]; present {
		target, err := e.encode(tgt)
		if err != nil {
			return nil, fmt.Errorf("call %q target: %w", fn, err)
		}
		return e.f.NewMemberCall(e.id(), fn, target, args...), nil
	}
	return e.f.NewCall(e.id(), fn, args...), nil
}

// parseCEL parses a CEL source fragment (a reference path or a raw expression) with macros disabled, so comprehension macros come back as plain call nodes that splice and unparse cleanly.
// The returned Expr keeps its own node ids; that is harmless because we unparse with an empty SourceInfo.
func (e *encoder) parseCEL(src string) (ast.Expr, error) {
	if len(src) > celMaxInputBytes {
		return nil, fmt.Errorf("CEL fragment exceeds maximum supported length of %d bytes", celMaxInputBytes)
	}
	p, err := parser.NewParser(parser.EnableOptionalSyntax(true))
	if err != nil {
		return nil, err
	}
	parsed, iss := p.Parse(common.NewTextSource(src))
	if len(iss.GetErrors()) > 0 {
		return nil, fmt.Errorf("%q is not a valid CEL expression: %s", src, iss.ToDisplayString())
	}
	return parsed.Expr(), nil
}

// parseReference parses an `ident` path and rejects anything that is not a pure reference (a bare identifier, field selection, or index chain, including the optional `?.` / `[?]` forms).
// This prevents a full expression fed through `ident` from being silently emitted; such input should use `raw`.
func (e *encoder) parseReference(src string) (ast.Expr, error) {
	expr, err := e.parseCEL(src)
	if err != nil {
		return nil, err
	}
	if !isReferencePath(expr) {
		return nil, fmt.Errorf("ident %q is not a reference path (identifier, field, or index chain); use { raw = %q } for a full expression", src, src)
	}
	return expr, nil
}

// validate confirms s is syntactically valid CEL under the standard environment.
func validate(s string) error {
	env, err := newParseEnv(false)
	if err != nil {
		return err
	}
	if _, iss := env.Parse(s); iss != nil && iss.Err() != nil {
		return iss.Err()
	}
	return nil
}

// IsValid reports whether s is a syntactically valid CEL expression. It never errors; malformed input returns false.
// With strict false it uses the lenient environment (optional types and two-variable comprehensions enabled), matching what real Kubernetes and GCP dialects accept.
// With strict true it uses base cel-go with no extensions, so optional-navigation syntax (?. / [?] / [?x] / {?k: v}) is rejected, which checks portability to a plain CEL host.
func IsValid(s string, strict bool) bool {
	if strict {
		env, err := celgo.NewEnv()
		if err != nil {
			return false
		}
		_, iss := env.Parse(s)
		return iss == nil || iss.Err() == nil
	}
	return validate(s) == nil
}

// Format parses a hand-written CEL string, failing on invalid input, and returns its canonical form, optionally reformatted by the given format options.
// Macro-call tracking is enabled so comprehension sugar (exists/map/…) round-trips.
func Format(s string, opts ...FormatOption) (string, error) {
	env, err := newParseEnv(true)
	if err != nil {
		return "", err
	}
	parsed, iss := env.Parse(s)
	if iss != nil && iss.Err() != nil {
		return "", iss.Err()
	}
	native := parsed.NativeRep()
	return formatExpr(native.Expr(), native.SourceInfo(), opts...)
}

// Encode builds a canonical CEL expression string from a decoded HCL data tree, validating the result so it can never return syntactically invalid CEL.
// The format options (empty by default) control wrapping and pretty-printing.
func Encode(node any, opts ...FormatOption) (string, error) {
	e := &encoder{f: ast.NewExprFactory()}
	expr, err := e.encode(node)
	if err != nil {
		return "", err
	}
	out, err := formatExpr(expr, ast.NewSourceInfo(nil), opts...)
	if err != nil {
		return "", err
	}
	if err := validate(out); err != nil {
		return "", fmt.Errorf("%w %q: %v", errInvalidOutput, out, err)
	}
	return out, nil
}

// errInvalidOutput marks the internal invariant break where the encoder somehow produced invalid CEL.
// It should be unreachable; the functions report it as an internal error, not an argument error.
var errInvalidOutput = errors.New("internal error: encoder produced invalid CEL")
