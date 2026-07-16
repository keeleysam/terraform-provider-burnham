package cel

import (
	"fmt"

	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/parser"
)

// Notations understood by Decode.
const (
	NotationCanonical = "canonical" // syntax.proto field names; operators as _==_ calls
	NotationStandard  = "standard"  // type-name keys + CEL operator tokens (the examples)
	NotationAliased   = "aliased"   // type-name keys + friendly word aliases
)

// fnToToken / fnToAlias reverse the operator table: a canonical operator function name maps back to its CEL surface token and its word alias.
var fnToToken = map[string]string{
	operators.Equals: "==", operators.NotEquals: "!=", operators.Less: "<", operators.LessEquals: "<=",
	operators.Greater: ">", operators.GreaterEquals: ">=", operators.LogicalAnd: "&&", operators.LogicalOr: "||",
	operators.LogicalNot: "!", operators.Negate: "-", operators.In: "in", operators.Conditional: "?:",
	operators.Add: "+", operators.Subtract: "-", operators.Multiply: "*", operators.Divide: "/", operators.Modulo: "%",
	operators.Index: "[]",
}

var fnToAlias = map[string]string{
	operators.Equals: "eq", operators.NotEquals: "ne", operators.Less: "lt", operators.LessEquals: "le",
	operators.Greater: "gt", operators.GreaterEquals: "ge", operators.LogicalAnd: "and", operators.LogicalOr: "or",
	operators.LogicalNot: "not", operators.Negate: "neg", operators.In: "in", operators.Conditional: "cond",
	operators.Add: "add", operators.Subtract: "sub", operators.Multiply: "mul", operators.Divide: "div", operators.Modulo: "mod",
	operators.Index: "index",
}

var fnKind = map[string]opKind{
	operators.Equals: kBinary, operators.NotEquals: kBinary, operators.Less: kBinary, operators.LessEquals: kBinary,
	operators.Greater: kBinary, operators.GreaterEquals: kBinary, operators.In: kBinary, operators.Index: kBinary,
	operators.Add: kBinary, operators.Subtract: kBinary, operators.Multiply: kBinary, operators.Divide: kBinary, operators.Modulo: kBinary,
	operators.LogicalAnd: kVariadic, operators.LogicalOr: kVariadic,
	operators.LogicalNot: kUnary, operators.Negate: kUnary,
	operators.Conditional: kTernary,
}

type decoder struct {
	si   *ast.SourceInfo
	mode string
}

// Decode parses a CEL string and returns the HCL data tree in the requested notation (canonical, standard, or aliased).
// It is the inverse of Encode.
func Decode(celString, notation string) (any, error) {
	switch notation {
	case NotationCanonical, NotationStandard, NotationAliased:
	case "":
		notation = NotationStandard
	default:
		return nil, fmt.Errorf("unknown notation %q; expected canonical, standard, or aliased", notation)
	}
	env, err := newParseEnv(true)
	if err != nil {
		return nil, err
	}
	parsed, iss := env.Parse(celString)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	native := parsed.NativeRep()
	d := &decoder{si: native.SourceInfo(), mode: notation}
	return d.decode(native.Expr()), nil
}

func (d *decoder) canonical() bool { return d.mode == NotationCanonical }

func (d *decoder) opKey(fn string) string {
	if d.mode == NotationAliased {
		return fnToAlias[fn]
	}
	return fnToToken[fn]
}

func (d *decoder) unparse(expr ast.Expr) string {
	s, err := parser.Unparse(expr, d.si)
	if err != nil {
		return ""
	}
	return s
}

func (d *decoder) decode(expr ast.Expr) any {
	// A macro (has/all/exists/map/filter/…) is recorded in source info as its call form; decode that instead of the raw comprehension.
	// Exception: in canonical notation, has() is emitted as its true node, a select_expr with test_only=true, since the CEL canonical AST has no has() call.
	// The comprehension macros stay in call form even in canonical notation because the fully-expanded comprehension_expr cannot be re-encoded.
	if mc, ok := d.si.GetMacroCall(expr.ID()); ok {
		hasPresenceTest := expr.Kind() == ast.SelectKind && expr.AsSelect().IsTestOnly()
		if !(d.canonical() && hasPresenceTest) {
			return d.decode(mc)
		}
	}

	switch expr.Kind() {
	case ast.LiteralKind:
		return d.decodeConst(expr)

	case ast.IdentKind:
		if d.canonical() {
			return map[string]any{"ident_expr": map[string]any{"name": expr.AsIdent()}}
		}
		return map[string]any{"ident": expr.AsIdent()}

	case ast.SelectKind:
		sel := expr.AsSelect()
		if !d.canonical() && !sel.IsTestOnly() && isReferencePath(expr) {
			return map[string]any{"ident": d.unparse(expr)}
		}
		se := map[string]any{"operand": d.decode(sel.Operand()), "field": sel.FieldName()}
		if sel.IsTestOnly() {
			se["test_only"] = true
		}
		return map[string]any{"select_expr": se}

	case ast.CallKind:
		return d.decodeCall(expr)

	case ast.ListKind:
		return d.decodeList(expr.AsList())

	case ast.MapKind:
		return d.decodeMap(expr.AsMap())

	case ast.StructKind:
		return d.decodeStruct(expr.AsStruct())

	default:
		// Anything we don't model (e.g. a bare comprehension with no macro call) round-trips via raw.
		return map[string]any{"raw": d.unparse(expr)}
	}
}

func (d *decoder) decodeCall(expr ast.Expr) any {
	// A pure reference path (incl. index / optional navigation) folds into a single ident in the surface notations.
	if !d.canonical() && isReferencePath(expr) {
		return map[string]any{"ident": d.unparse(expr)}
	}
	call := expr.AsCall()
	fn := call.FunctionName()

	// In canonical notation, has() is the presence-test select node, never a call, even when it appears nested inside a comprehension predicate (where the macro-call form carries it as a has() call).
	if d.canonical() && !call.IsMemberFunction() && fn == operators.Has && len(call.Args()) == 1 {
		if arg := call.Args()[0]; arg.Kind() == ast.SelectKind {
			sel := arg.AsSelect()
			return map[string]any{"select_expr": map[string]any{
				"operand":   d.decode(sel.Operand()),
				"field":     sel.FieldName(),
				"test_only": true,
			}}
		}
	}

	if !call.IsMemberFunction() {
		if _, isOp := fnKind[fn]; isOp {
			if d.canonical() {
				return map[string]any{"call_expr": map[string]any{"function": fn, "args": d.decodeArgs(call.Args())}}
			}
			return d.decodeOperator(fn, call.Args())
		}
	}

	spec := map[string]any{"function": fn}
	if call.IsMemberFunction() {
		spec["target"] = d.decode(call.Target())
	}
	if len(call.Args()) > 0 {
		spec["args"] = d.decodeArgs(call.Args())
	}
	if d.canonical() {
		return map[string]any{"call_expr": spec}
	}
	return map[string]any{"call": spec}
}

func (d *decoder) decodeOperator(fn string, args []ast.Expr) any {
	key := d.opKey(fn)
	switch fnKind[fn] {
	case kUnary:
		operand := d.decode(args[0])
		// A unary operator with a list-literal operand must not be emitted as a bare list under the operator key.
		// In the surface notations that bare list reads as the operand sequence and gets spread, corrupting "-[1, 2]" into "1 - 2" and making "![a, b]" an arity error.
		// Wrap it as an explicit list_expr node so encode keeps it as the single operand.
		if _, isList := operand.([]any); isList {
			operand = map[string]any{"list_expr": map[string]any{"elements": operand}}
		}
		return map[string]any{key: operand}
	case kTernary:
		return map[string]any{key: d.decodeArgs(args)}
	case kVariadic:
		// Flatten a left-associative chain of the same operator into one list.
		return map[string]any{key: d.flatten(fn, args)}
	default: // kBinary
		return map[string]any{key: d.decodeArgs(args)}
	}
}

func (d *decoder) flatten(fn string, args []ast.Expr) []any {
	var out []any
	for _, a := range args {
		if a.Kind() == ast.CallKind {
			c := a.AsCall()
			if !c.IsMemberFunction() && c.FunctionName() == fn {
				if _, hasMacro := d.si.GetMacroCall(a.ID()); !hasMacro {
					out = append(out, d.flatten(fn, c.Args())...)
					continue
				}
			}
		}
		out = append(out, d.decode(a))
	}
	return out
}

func (d *decoder) decodeArgs(args []ast.Expr) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = d.decode(a)
	}
	return out
}

func (d *decoder) decodeList(lst ast.ListExpr) any {
	elems := lst.Elements()
	optional := make(map[int]bool)
	for _, i := range lst.OptionalIndices() {
		optional[int(i)] = true
	}
	if d.canonical() {
		out := make([]any, len(elems))
		for i, el := range elems {
			out[i] = d.decode(el)
		}
		le := map[string]any{"elements": out}
		if len(optional) > 0 {
			idx := make([]any, 0, len(optional))
			for _, i := range lst.OptionalIndices() {
				idx = append(idx, int(i))
			}
			le["optional_indices"] = idx
		}
		return map[string]any{"list_expr": le}
	}
	out := make([]any, len(elems))
	for i, el := range elems {
		dec := d.decode(el)
		if optional[i] {
			dec = map[string]any{"optional": dec}
		}
		out[i] = dec
	}
	return out
}

func (d *decoder) decodeMap(mp ast.MapExpr) any {
	entries := make([]any, 0, mp.Size())
	for _, e := range mp.Entries() {
		me := e.AsMapEntry()
		entry := map[string]any{"map_key": d.decode(me.Key()), "value": d.decode(me.Value())}
		if me.IsOptional() {
			entry["optional_entry"] = true
		}
		entries = append(entries, entry)
	}
	return map[string]any{"struct_expr": map[string]any{"entries": entries}}
}

func (d *decoder) decodeStruct(st ast.StructExpr) any {
	entries := make([]any, 0, len(st.Fields()))
	for _, e := range st.Fields() {
		sf := e.AsStructField()
		entry := map[string]any{"field_key": sf.Name(), "value": d.decode(sf.Value())}
		if sf.IsOptional() {
			entry["optional_entry"] = true
		}
		entries = append(entries, entry)
	}
	return map[string]any{"struct_expr": map[string]any{"message_name": st.TypeName(), "entries": entries}}
}

func (d *decoder) decodeConst(expr ast.Expr) any {
	switch cv := expr.AsLiteral().(type) {
	case types.Bool:
		return d.constNode("bool_value", bool(cv), bool(cv))
	case types.Int:
		return d.constNode("int64_value", int64(cv), int64(cv))
	case types.Uint:
		return d.constNode("uint64_value", uint64(cv), map[string]any{"const": map[string]any{"uint64_value": uint64(cv)}})
	case types.Double:
		return d.constNode("double_value", float64(cv), map[string]any{"const": map[string]any{"double_value": float64(cv)}})
	case types.String:
		return d.constNode("string_value", string(cv), string(cv))
	case types.Null:
		return d.constNode("null_value", nil, nil)
	default:
		// Bytes (and any other literal) round-trip via raw: cel-go unparses a bytes literal to octal-escaped ASCII (b"\377"), which is JSON-safe.
		// A structured bytes string would be corrupted crossing the JSON boundary if it held non-UTF-8 bytes.
		return map[string]any{"raw": d.unparse(expr)}
	}
}

// constNode returns the canonical const_expr form or the surface literal form.
// `canonicalVal` is the value under the constant_kind key; `surface` is the bare or { const = ... } surface representation.
func (d *decoder) constNode(kind string, canonicalVal any, surface any) any {
	if d.canonical() {
		return map[string]any{"const_expr": map[string]any{kind: canonicalVal}}
	}
	return surface
}
