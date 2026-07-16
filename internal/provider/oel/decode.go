package oel

import (
	"fmt"
	"strings"

	okta "github.com/keeleysam/okta-expression-parser"
)

// Decode parses an Okta EL string and returns the surface-notation data tree that oelencode consumes, so oelencode(oeldecode(x)) round-trips to the canonical form of x.
func Decode(s string) (any, error) {
	if err := checkNestingDepth(s); err != nil {
		return nil, err
	}
	n, err := okta.New().Parse(s)
	if err != nil {
		return nil, err
	}
	return decodeNode(n), nil
}

// decodeNode converts one parser AST node into the surface notation. Anything without a dedicated surface form (currently only a dotted path that embeds a member-of method hop) falls back to a { raw = "<canonical>" } escape, which oelencode re-parses, so every node still round-trips.
func decodeNode(n okta.Node) any {
	switch v := n.(type) {
	case okta.Literal:
		return v.Value
	case okta.ArrayLit:
		return decodeList(v.Elements)
	case okta.CommaList:
		return decodeList(v.Elements)
	case okta.PathExpr:
		return decodePath(v)
	case okta.Comparison:
		return m(v.Op, []any{decodeNode(v.Left), decodeNode(v.Right)})
	case okta.Additive:
		return m("+", []any{decodeNode(v.Left), decodeNode(v.Right)})
	case okta.Logical:
		return m(v.Op, decodeList(v.Operands))
	case okta.Not:
		return m("!", decodeNode(v.Operand))
	case okta.Ternary:
		return m("cond", []any{decodeNode(v.Cond), decodeNode(v.True), decodeNode(v.False)})
	case okta.Elvis:
		return m("elvis", []any{decodeNode(v.Left), decodeNode(v.Right)})
	case okta.MatchesExpr:
		return m("matches", []any{decodeNode(v.Left), decodeNode(v.Right)})
	case okta.ClassCall:
		return m("call", withArgs(map[string]any{"class": v.Class, "method": v.Method}, v.Arg))
	case okta.MemberOfExpr:
		return m("call", withArgs(map[string]any{"function": memberOfName(v.Kind)}, v.Arg))
	case okta.FuncCall:
		return m("call", withArgs(map[string]any{"function": v.Name}, v.Args))
	case okta.MethodCall:
		return m("call", withArgs(map[string]any{"target": decodeNode(v.Target), "method": v.Method}, v.Args))
	case okta.FieldAccess:
		return m("select", map[string]any{"operand": decodeNode(v.Target), "field": v.Field})
	case okta.IndexExpr:
		return m("index", map[string]any{"base": decodeNode(v.Base), "index": decodeNode(v.Index)})
	case okta.Projection:
		return m("project", map[string]any{"base": decodeNode(v.Base), "expr": decodeNode(v.Expr)})
	case okta.MapLit:
		return decodeMap(v)
	default:
		return m("raw", n.String())
	}
}

func decodeList(nodes []okta.Node) []any {
	out := make([]any, len(nodes))
	for i, n := range nodes {
		out[i] = decodeNode(n)
	}
	return out
}

// decodePath renders a pure dotted path as { ident = "..." }. A path that carries a member-of method hop has no direct surface form, so it falls back to raw.
func decodePath(p okta.PathExpr) any {
	var b strings.Builder
	if p.RootUser {
		b.WriteString("user")
	} else {
		b.WriteString(p.RootName)
	}
	for _, hop := range p.Hops {
		nh, ok := hop.(okta.NameHop)
		if !ok {
			return m("raw", p.String())
		}
		b.WriteByte('.')
		b.WriteString(nh.Name)
	}
	return m("ident", b.String())
}

// decodeMap renders a MapLit as { map = [ { key = ..., value = ... } ] }. A non-string key (unusual) falls back to raw for the whole map.
func decodeMap(ml okta.MapLit) any {
	entries := make([]any, 0, len(ml.Entries))
	for _, e := range ml.Entries {
		lit, ok := e.Key.(okta.Literal)
		if !ok {
			return m("raw", ml.String())
		}
		key, ok := lit.Value.(string)
		if !ok {
			return m("raw", ml.String())
		}
		entries = append(entries, map[string]any{"key": key, "value": decodeNode(e.Value)})
	}
	return m("map", entries)
}

// withArgs adds an "args" list to a call spec when arg is non-nil, flattening a CommaList into multiple args.
func withArgs(spec map[string]any, arg okta.Node) map[string]any {
	if arg == nil {
		return spec
	}
	if cl, ok := arg.(okta.CommaList); ok {
		spec["args"] = decodeList(cl.Elements)
	} else {
		spec["args"] = []any{decodeNode(arg)}
	}
	return spec
}

// m builds a single-key node object, the surface-notation building block (also used by the package tests).
func m(k string, v any) map[string]any { return map[string]any{k: v} }

// memberOfName is the builtin name for a MemberOfKind, the inverse of the encoder's memberOfKinds map.
func memberOfName(kind okta.MemberOfKind) string {
	for name, k := range memberOfKinds {
		if k == kind {
			return name
		}
	}
	return fmt.Sprintf("isMemberOfGroup(unknown kind %d)", kind)
}
