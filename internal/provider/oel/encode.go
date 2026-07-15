package oel

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	okta "github.com/keeleysam/okta-expression-parser"
)

// errInvalidOutput signals that a built expression failed to parse back, i.e. we would otherwise have emitted syntactically invalid OEL. It wraps the offending output so callers can attribute the failure.
var errInvalidOutput = errors.New("built expression is not valid Okta EL")

// encoder builds one expression.
type encoder struct{}

// Encode builds a canonical Okta EL string from an HCL data tree.
//
// The output is parsed back before returning, so Encode never emits syntactically invalid OEL. It builds the parser's own AST nodes throughout, so the result is byte-identical to the canonical form oelformat produces.
func Encode(node any) (string, error) {
	e := &encoder{}
	n, err := e.build(node)
	if err != nil {
		return "", err
	}
	out := n.String()
	if _, perr := okta.New().Parse(out); perr != nil {
		return "", fmt.Errorf("%w %q: %v", errInvalidOutput, out, perr)
	}
	return out, nil
}

// build converts one HCL data node into an okta.Node.
func (e *encoder) build(node any) (okta.Node, error) {
	switch v := node.(type) {
	case nil:
		return okta.Literal{Value: nil}, nil
	case bool:
		return okta.Literal{Value: v}, nil
	case string:
		return okta.Literal{Value: v}, nil
	case int:
		return okta.Literal{Value: v}, nil
	case int64:
		return okta.Literal{Value: int(v)}, nil
	case float64:
		return numberLiteral(strconv.FormatFloat(v, 'g', -1, 64))
	case []any:
		elems, err := e.buildList(v)
		if err != nil {
			return nil, err
		}
		return okta.ArrayLit{Elements: elems}, nil
	case map[string]any:
		return e.buildObject(v)
	default:
		// json.Number arrives as a Stringer over the numeric text.
		if s, ok := node.(fmt.Stringer); ok {
			return numberLiteral(s.String())
		}
		return nil, fmt.Errorf("unsupported value type %T", node)
	}
}

func (e *encoder) buildList(items []any) ([]okta.Node, error) {
	out := make([]okta.Node, len(items))
	for i, item := range items {
		el, err := e.build(item)
		if err != nil {
			return nil, fmt.Errorf("element %d: %w", i, err)
		}
		out[i] = el
	}
	return out, nil
}

// numberLiteral turns numeric text into an int Literal when it is integral, else a float64 Literal, so output matches how the number was written.
func numberLiteral(s string) (okta.Node, error) {
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.Atoi(s); err == nil {
			return okta.Literal{Value: i}, nil
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q: %w", s, err)
	}
	return okta.Literal{Value: f}, nil
}

// comparisonOps maps surface tokens and friendly aliases to the canonical comparison operator.
var comparisonOps = map[string]string{
	"==": "==", "!=": "!=", "<": "<", ">": ">", "<=": "<=", ">=": ">=",
	"eq": "==", "ne": "!=", "lt": "<", "gt": ">", "le": "<=", "ge": ">=",
}

// logicalOps maps surface tokens and aliases (word and symbolic) to the canonical AND/OR keyword.
var logicalOps = map[string]string{
	"AND": "AND", "and": "AND", "&&": "AND",
	"OR": "OR", "or": "OR", "||": "OR",
}

// buildObject dispatches a single-key node object to the construct its key names.
func (e *encoder) buildObject(m map[string]any) (okta.Node, error) {
	if len(m) != 1 {
		return nil, fmt.Errorf("node object must have exactly one key, got %d", len(m))
	}
	for k, v := range m {
		switch {
		case k == "ident":
			return buildIdent(v)
		case k == "raw":
			return buildRaw(v)
		case k == "call":
			return e.buildCall(v)
		case k == "select":
			return e.buildSelect(v)
		case k == "index":
			return e.buildIndex(v)
		case k == "project":
			return e.buildProjection(v)
		case k == "map":
			return e.buildMap(v)
		case k == "!" || k == "not":
			operand, err := e.build(v)
			if err != nil {
				return nil, fmt.Errorf("%q operand: %w", k, err)
			}
			return okta.Not{Operand: operand}, nil
		case comparisonOps[k] != "":
			return e.buildComparison(comparisonOps[k], v)
		case logicalOps[k] != "":
			return e.buildLogical(logicalOps[k], v)
		case k == "+":
			return e.buildAdditive(v)
		case k == "?:" || k == "cond":
			return e.buildTernary(v)
		case k == "elvis":
			return e.buildElvis(v)
		case k == "matches":
			return e.buildMatches(v)
		default:
			return nil, fmt.Errorf("unknown node key %q", k)
		}
	}
	return nil, fmt.Errorf("empty node object")
}

// operands builds each element of an operator's operand list, requiring a list value.
func (e *encoder) operands(v any) ([]okta.Node, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("operator operands must be a list, got %T", v)
	}
	out := make([]okta.Node, len(list))
	for i, item := range list {
		n, err := e.build(item)
		if err != nil {
			return nil, fmt.Errorf("operand %d: %w", i, err)
		}
		out[i] = n
	}
	return out, nil
}

func (e *encoder) buildComparison(op string, v any) (okta.Node, error) {
	ops, err := e.operands(v)
	if err != nil {
		return nil, err
	}
	if len(ops) != 2 {
		return nil, fmt.Errorf("comparison %q needs 2 operands, got %d", op, len(ops))
	}
	return okta.Comparison{Op: op, Left: ops[0], Right: ops[1]}, nil
}

func (e *encoder) buildLogical(op string, v any) (okta.Node, error) {
	ops, err := e.operands(v)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, fmt.Errorf("logical %q needs at least 1 operand", op)
	}
	if len(ops) == 1 {
		return ops[0], nil
	}
	return okta.Logical{Op: op, Operands: ops}, nil
}

// buildAdditive folds a "+" chain left-associatively, since okta.Additive is binary.
func (e *encoder) buildAdditive(v any) (okta.Node, error) {
	ops, err := e.operands(v)
	if err != nil {
		return nil, err
	}
	if len(ops) < 2 {
		return nil, fmt.Errorf(`"+" needs at least 2 operands, got %d`, len(ops))
	}
	acc := ops[0]
	for _, next := range ops[1:] {
		acc = okta.Additive{Left: acc, Right: next}
	}
	return acc, nil
}

func (e *encoder) buildTernary(v any) (okta.Node, error) {
	ops, err := e.operands(v)
	if err != nil {
		return nil, err
	}
	if len(ops) != 3 {
		return nil, fmt.Errorf("ternary needs 3 operands (cond, true, false), got %d", len(ops))
	}
	return okta.Ternary{Cond: ops[0], True: ops[1], False: ops[2]}, nil
}

// memberOfKinds maps the bare group-membership builtin names to their upstream MemberOfKind.
var memberOfKinds = map[string]okta.MemberOfKind{
	"isMemberOfGroup":               okta.MemberOf,
	"isMemberOfAnyGroup":            okta.MemberOfAny,
	"isMemberOfGroupName":           okta.MemberOfName,
	"isMemberOfGroupNameStartsWith": okta.MemberOfGroupStartsWith,
	"isMemberOfGroupNameContains":   okta.MemberOfGroupContains,
	"isMemberOfGroupNameRegex":      okta.MemberOfGroupNameRegex,
}

// buildCall handles a call node in three forms: a namespaced class method (class + method), a bare function (function; a group-membership builtin maps to the upstream node, any other name to our funcCall extension), or a receiver method call (target + method, an extension).
func (e *encoder) buildCall(v any) (okta.Node, error) {
	spec, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("call must be an object, got %T", v)
	}
	for k := range spec {
		if k != "class" && k != "method" && k != "function" && k != "target" && k != "args" {
			return nil, fmt.Errorf("unknown call key %q; expected class, method, function, target, or args", k)
		}
	}
	_, hasClass := spec["class"]
	_, hasFunc := spec["function"]
	_, hasTarget := spec["target"]
	forms := 0
	for _, has := range []bool{hasClass, hasFunc, hasTarget} {
		if has {
			forms++
		}
	}
	if forms != 1 {
		return nil, fmt.Errorf("call must have exactly one of class/method, function, or target/method")
	}

	args, err := e.buildArgs(spec["args"])
	if err != nil {
		return nil, err
	}

	switch {
	case hasClass:
		class, _ := spec["class"].(string)
		method, _ := spec["method"].(string)
		if class == "" || method == "" {
			return nil, fmt.Errorf("class call requires non-empty class and method")
		}
		return okta.ClassCall{Class: class, Method: method, Arg: argsToNode(args)}, nil
	case hasTarget:
		method, _ := spec["method"].(string)
		if method == "" {
			return nil, fmt.Errorf("method call requires a non-empty method")
		}
		target, err := e.build(spec["target"])
		if err != nil {
			return nil, fmt.Errorf("call target: %w", err)
		}
		return okta.MethodCall{Target: target, Method: method, Args: argsToNode(args)}, nil
	default: // hasFunc
		fn, _ := spec["function"].(string)
		if fn == "" {
			return nil, fmt.Errorf("function call requires a non-empty function name")
		}
		if kind, known := memberOfKinds[fn]; known {
			if len(args) == 0 {
				return nil, fmt.Errorf("%s requires at least one argument", fn)
			}
			return okta.MemberOfExpr{Kind: kind, Arg: argsToNode(args)}, nil
		}
		return okta.FuncCall{Name: fn, Args: argsToNode(args)}, nil
	}
}

// buildArgs builds a call's argument list; a missing args key means no arguments.
func (e *encoder) buildArgs(v any) ([]okta.Node, error) {
	if v == nil {
		return nil, nil
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("call args must be a list, got %T", v)
	}
	out := make([]okta.Node, len(list))
	for i, item := range list {
		n, err := e.build(item)
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		out[i] = n
	}
	return out, nil
}

// argsToNode collapses an argument list into the single Arg the upstream nodes expect: nil for none, the sole node for one, a CommaList for several.
func argsToNode(args []okta.Node) okta.Node {
	switch len(args) {
	case 0:
		return nil
	case 1:
		return args[0]
	default:
		return okta.CommaList{Elements: args}
	}
}

// buildIdent turns a dotted reference string ("user.city", "appuser.email") into a PathExpr.
func buildIdent(v any) (okta.Node, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("ident must be a string, got %T", v)
	}
	if s == "" {
		return nil, fmt.Errorf("ident must not be empty")
	}
	parts := strings.Split(s, ".")
	p := okta.PathExpr{}
	if parts[0] == "user" {
		p.RootUser = true
	} else {
		p.RootName = parts[0]
	}
	for _, name := range parts[1:] {
		p.Hops = append(p.Hops, okta.NameHop{Name: name})
	}
	return p, nil
}

// buildRaw embeds a hand-written Okta EL fragment: the string is parsed (so it is validated and normalized) and its node used directly.
func buildRaw(v any) (okta.Node, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("raw must be a string, got %T", v)
	}
	n, err := okta.New().Parse(s)
	if err != nil {
		return nil, fmt.Errorf("raw is not valid Okta EL: %w", err)
	}
	return n, nil
}

// subNode builds one named sub-node from a spec object, erroring if it is absent.
func (e *encoder) subNode(spec map[string]any, key string) (okta.Node, error) {
	v, ok := spec[key]
	if !ok {
		return nil, fmt.Errorf("missing %q", key)
	}
	return e.build(v)
}

func objectSpec(v any, allowed ...string) (map[string]any, error) {
	spec, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected an object, got %T", v)
	}
	for k := range spec {
		if !contains(allowed, k) {
			return nil, fmt.Errorf("unknown key %q; expected one of %s", k, strings.Join(allowed, ", "))
		}
	}
	return spec, nil
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// buildSelect emits a field access on an arbitrary receiver ("<operand>.<field>"), used when the receiver is not a plain identifier path (e.g. after a function call). Plain paths should use ident.
func (e *encoder) buildSelect(v any) (okta.Node, error) {
	spec, err := objectSpec(v, "operand", "field")
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	field, _ := spec["field"].(string)
	if field == "" {
		return nil, fmt.Errorf("select requires a non-empty field")
	}
	operand, err := e.subNode(spec, "operand")
	if err != nil {
		return nil, fmt.Errorf("select operand: %w", err)
	}
	return okta.FieldAccess{Target: operand, Field: field}, nil
}

// buildIndex emits an index access ("<base>[<index>]").
func (e *encoder) buildIndex(v any) (okta.Node, error) {
	spec, err := objectSpec(v, "base", "index")
	if err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}
	base, err := e.subNode(spec, "base")
	if err != nil {
		return nil, fmt.Errorf("index base: %w", err)
	}
	idx, err := e.subNode(spec, "index")
	if err != nil {
		return nil, fmt.Errorf("index index: %w", err)
	}
	return okta.IndexExpr{Base: base, Index: idx}, nil
}

// buildProjection emits a collection projection ("<base>.![<expr>]").
func (e *encoder) buildProjection(v any) (okta.Node, error) {
	spec, err := objectSpec(v, "base", "expr")
	if err != nil {
		return nil, fmt.Errorf("project: %w", err)
	}
	base, err := e.subNode(spec, "base")
	if err != nil {
		return nil, fmt.Errorf("project base: %w", err)
	}
	expr, err := e.subNode(spec, "expr")
	if err != nil {
		return nil, fmt.Errorf("project expr: %w", err)
	}
	return okta.Projection{Base: base, Expr: expr}, nil
}

// buildElvis emits the Elvis (null-coalescing) operator ("<a> ?: <b>").
func (e *encoder) buildElvis(v any) (okta.Node, error) {
	ops, err := e.operands(v)
	if err != nil {
		return nil, err
	}
	if len(ops) != 2 {
		return nil, fmt.Errorf("elvis needs 2 operands, got %d", len(ops))
	}
	return okta.Elvis{Left: ops[0], Right: ops[1]}, nil
}

// buildMatches emits the regex matches operator ("<a> matches <b>").
func (e *encoder) buildMatches(v any) (okta.Node, error) {
	ops, err := e.operands(v)
	if err != nil {
		return nil, err
	}
	if len(ops) != 2 {
		return nil, fmt.Errorf("matches needs 2 operands, got %d", len(ops))
	}
	return okta.MatchesExpr{Left: ops[0], Right: ops[1]}, nil
}

// buildMap emits an OEL map literal ("{'k1': v1, 'k2': v2}") from an ordered list of {key, value} entries. Order is preserved so output is deterministic.
func (e *encoder) buildMap(v any) (okta.Node, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("map must be a list of {key, value} entries, got %T", v)
	}
	entries := make([]okta.MapEntry, 0, len(list))
	for i, item := range list {
		spec, err := objectSpec(item, "key", "value")
		if err != nil {
			return nil, fmt.Errorf("map entry %d: %w", i, err)
		}
		key, ok := spec["key"].(string)
		if !ok || key == "" {
			return nil, fmt.Errorf("map entry %d: key must be a non-empty string", i)
		}
		val, err := e.subNode(spec, "value")
		if err != nil {
			return nil, fmt.Errorf("map entry %d value: %w", i, err)
		}
		entries = append(entries, okta.MapEntry{Key: okta.Literal{Value: key}, Value: val})
	}
	return okta.MapLit{Entries: entries}, nil
}
