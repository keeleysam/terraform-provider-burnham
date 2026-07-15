package promql

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// errInvalidOutput signals that the data tree did not describe a valid PromQL expression, so no query could be built from it.
var errInvalidOutput = errors.New("built expression is not valid PromQL")

// Encode builds a canonical PromQL query string from an HCL data tree.
//
// The tree is modeled on the Prometheus AST, using the node types the experimental /api/v1/parse_query endpoint exposes: each construct is a single-key object naming a node type, with the node's fields inside (not parse_query's literal wire format). A bare number is a numberLiteral and a bare string is a stringLiteral. The parser's own AST is built and re-serialized, so the output is canonical (byte-identical to what Format produces) and Encode never emits an invalid query.
func Encode(tree any) (string, error) {
	e, err := build(tree)
	if err != nil {
		return "", err
	}
	out := e.String()
	if _, perr := parser.NewParser(parser.Options{}).ParseExpr(out); perr != nil {
		return "", fmt.Errorf("%w %q: %v", errInvalidOutput, out, perr)
	}
	return out, nil
}

func build(node any) (parser.Expr, error) {
	switch v := node.(type) {
	case string:
		return &parser.StringLiteral{Val: v}, nil
	case int:
		return &parser.NumberLiteral{Val: float64(v)}, nil
	case int64:
		return &parser.NumberLiteral{Val: float64(v)}, nil
	case float64:
		return &parser.NumberLiteral{Val: v}, nil
	case map[string]any:
		return buildObject(v)
	default:
		if s, ok := node.(fmt.Stringer); ok { // json.Number
			f, err := strconv.ParseFloat(s.String(), 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q: %w", s.String(), err)
			}
			return &parser.NumberLiteral{Val: f}, nil
		}
		return nil, fmt.Errorf("unsupported value type %T", node)
	}
}

func buildObject(m map[string]any) (parser.Expr, error) {
	if len(m) != 1 {
		return nil, fmt.Errorf("node object must have exactly one key, got %d", len(m))
	}
	for k, v := range m {
		switch k {
		case "vectorSelector":
			return buildVectorSelector(v)
		case "matrixSelector":
			return buildMatrixSelector(v)
		case "call":
			return buildCall(v)
		case "aggregation":
			return buildAggregation(v)
		case "binaryExpr":
			return buildBinary(v)
		case "subquery":
			return buildSubquery(v)
		case "paren":
			inner, err := build(v)
			if err != nil {
				return nil, fmt.Errorf("paren: %w", err)
			}
			return &parser.ParenExpr{Expr: inner}, nil
		case "neg":
			inner, err := build(v)
			if err != nil {
				return nil, fmt.Errorf("neg: %w", err)
			}
			return &parser.UnaryExpr{Op: parser.SUB, Expr: inner}, nil
		case "pos":
			inner, err := build(v)
			if err != nil {
				return nil, fmt.Errorf("pos: %w", err)
			}
			return &parser.UnaryExpr{Op: parser.ADD, Expr: inner}, nil
		case "raw":
			return buildRaw(v)
		default:
			return nil, fmt.Errorf("unknown node key %q", k)
		}
	}
	return nil, fmt.Errorf("empty node object")
}

// buildRaw embeds a hand-written PromQL fragment: it is parsed (and so validated) and its AST used directly.
func buildRaw(v any) (parser.Expr, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("raw must be a string, got %T", v)
	}
	e, err := parser.NewParser(parser.Options{}).ParseExpr(s)
	if err != nil {
		return nil, fmt.Errorf("raw is not valid PromQL: %w", err)
	}
	return e, nil
}

// vectorSelectorFields builds the shared selector fields (name, matchers, offset, at), used by both vectorSelector and matrixSelector.
func vectorSelectorFields(spec map[string]any) (*parser.VectorSelector, error) {
	vs := &parser.VectorSelector{}
	if name, ok := spec["name"].(string); ok {
		vs.Name = name
	}
	if raw, ok := spec["matchers"]; ok && raw != nil {
		ms, err := buildMatchers(raw)
		if err != nil {
			return nil, err
		}
		vs.LabelMatchers = ms
	}
	if off, ok := spec["offset"]; ok && off != nil {
		d, err := duration(off)
		if err != nil {
			return nil, fmt.Errorf("offset: %w", err)
		}
		vs.OriginalOffset = d
	}
	if at, ok := spec["at"]; ok && at != nil {
		ts, soe, err := parseAt(at)
		if err != nil {
			return nil, err
		}
		vs.Timestamp, vs.StartOrEnd = ts, soe
	}
	return vs, nil
}

func buildVectorSelector(v any) (parser.Expr, error) {
	spec, err := objectSpec(v, "name", "matchers", "offset", "at")
	if err != nil {
		return nil, fmt.Errorf("vectorSelector: %w", err)
	}
	return vectorSelectorFields(spec)
}

func buildMatrixSelector(v any) (parser.Expr, error) {
	spec, err := objectSpec(v, "name", "matchers", "offset", "at", "range")
	if err != nil {
		return nil, fmt.Errorf("matrixSelector: %w", err)
	}
	vs, err := vectorSelectorFields(spec)
	if err != nil {
		return nil, err
	}
	rng, ok := spec["range"]
	if !ok || rng == nil {
		return nil, fmt.Errorf("matrixSelector requires a range")
	}
	d, err := duration(rng)
	if err != nil {
		return nil, fmt.Errorf("matrixSelector range: %w", err)
	}
	return &parser.MatrixSelector{VectorSelector: vs, Range: d}, nil
}

func buildCall(v any) (parser.Expr, error) {
	spec, err := objectSpec(v, "func", "args")
	if err != nil {
		return nil, fmt.Errorf("call: %w", err)
	}
	name, _ := spec["func"].(string)
	fn, ok := parser.Functions[name]
	if !ok {
		return nil, fmt.Errorf("unknown PromQL function %q", name)
	}
	if fn.Experimental {
		return nil, fmt.Errorf("PromQL function %q is experimental and not supported", name)
	}
	args, err := buildArgs(spec["args"])
	if err != nil {
		return nil, err
	}
	return &parser.Call{Func: fn, Args: args}, nil
}

func buildAggregation(v any) (parser.Expr, error) {
	spec, err := objectSpec(v, "op", "expr", "param", "by", "without")
	if err != nil {
		return nil, fmt.Errorf("aggregation: %w", err)
	}
	opName, _ := spec["op"].(string)
	op, ok := aggOps[opName]
	if !ok {
		return nil, fmt.Errorf("unknown aggregation operator %q", opName)
	}
	if _, ok := spec["expr"]; !ok {
		return nil, fmt.Errorf("aggregation requires an expr")
	}
	expr, err := build(spec["expr"])
	if err != nil {
		return nil, fmt.Errorf("aggregation expr: %w", err)
	}
	agg := &parser.AggregateExpr{Op: op, Expr: expr}
	_, hasBy := spec["by"]
	_, hasWithout := spec["without"]
	if hasBy && hasWithout {
		return nil, fmt.Errorf("aggregation takes at most one of by or without")
	}
	if hasBy {
		if agg.Grouping, err = stringList(spec["by"]); err != nil {
			return nil, fmt.Errorf("aggregation by: %w", err)
		}
	} else if hasWithout {
		agg.Without = true
		if agg.Grouping, err = stringList(spec["without"]); err != nil {
			return nil, fmt.Errorf("aggregation without: %w", err)
		}
	}
	if p, ok := spec["param"]; ok && p != nil {
		if agg.Param, err = build(p); err != nil {
			return nil, fmt.Errorf("aggregation param: %w", err)
		}
	}
	return agg, nil
}

func buildBinary(v any) (parser.Expr, error) {
	spec, err := objectSpec(v, "op", "lhs", "rhs", "bool", "on", "ignoring", "group_left", "group_right")
	if err != nil {
		return nil, fmt.Errorf("binaryExpr: %w", err)
	}
	opName, _ := spec["op"].(string)
	op, ok := binaryOps[opName]
	if !ok {
		return nil, fmt.Errorf("unknown binary operator %q", opName)
	}
	lhs, err := buildChild(spec, "lhs")
	if err != nil {
		return nil, err
	}
	rhs, err := buildChild(spec, "rhs")
	if err != nil {
		return nil, err
	}
	bin := &parser.BinaryExpr{Op: op, LHS: lhs, RHS: rhs}
	if b, ok := spec["bool"].(bool); ok {
		bin.ReturnBool = b
	}
	vm, err := buildVectorMatching(spec)
	if err != nil {
		return nil, err
	}
	bin.VectorMatching = vm
	return bin, nil
}

func buildVectorMatching(spec map[string]any) (*parser.VectorMatching, error) {
	_, hasOn := spec["on"]
	_, hasIgnoring := spec["ignoring"]
	_, hasGL := spec["group_left"]
	_, hasGR := spec["group_right"]
	if !hasOn && !hasIgnoring && !hasGL && !hasGR {
		return nil, nil
	}
	if hasOn && hasIgnoring {
		return nil, fmt.Errorf("binaryExpr matching takes at most one of on or ignoring")
	}
	if hasGL && hasGR {
		return nil, fmt.Errorf("binaryExpr matching takes at most one of group_left or group_right")
	}
	vm := &parser.VectorMatching{Card: parser.CardOneToOne}
	var err error
	if hasOn {
		vm.On = true
		if vm.MatchingLabels, err = stringList(spec["on"]); err != nil {
			return nil, fmt.Errorf("on: %w", err)
		}
	} else if hasIgnoring {
		if vm.MatchingLabels, err = stringList(spec["ignoring"]); err != nil {
			return nil, fmt.Errorf("ignoring: %w", err)
		}
	}
	if hasGL {
		vm.Card = parser.CardManyToOne
		if vm.Include, err = stringList(spec["group_left"]); err != nil {
			return nil, fmt.Errorf("group_left: %w", err)
		}
	} else if hasGR {
		vm.Card = parser.CardOneToMany
		if vm.Include, err = stringList(spec["group_right"]); err != nil {
			return nil, fmt.Errorf("group_right: %w", err)
		}
	}
	return vm, nil
}

func buildSubquery(v any) (parser.Expr, error) {
	spec, err := objectSpec(v, "expr", "range", "step", "offset", "at")
	if err != nil {
		return nil, fmt.Errorf("subquery: %w", err)
	}
	if _, ok := spec["expr"]; !ok {
		return nil, fmt.Errorf("subquery requires an expr")
	}
	expr, err := build(spec["expr"])
	if err != nil {
		return nil, fmt.Errorf("subquery expr: %w", err)
	}
	rng, ok := spec["range"]
	if !ok || rng == nil {
		return nil, fmt.Errorf("subquery requires a range")
	}
	d, err := duration(rng)
	if err != nil {
		return nil, fmt.Errorf("subquery range: %w", err)
	}
	sq := &parser.SubqueryExpr{Expr: expr, Range: d}
	if step, ok := spec["step"]; ok && step != nil {
		if sq.Step, err = duration(step); err != nil {
			return nil, fmt.Errorf("subquery step: %w", err)
		}
	}
	if off, ok := spec["offset"]; ok && off != nil {
		if sq.OriginalOffset, err = duration(off); err != nil {
			return nil, fmt.Errorf("subquery offset: %w", err)
		}
	}
	if at, ok := spec["at"]; ok && at != nil {
		ts, soe, aerr := parseAt(at)
		if aerr != nil {
			return nil, fmt.Errorf("subquery %w", aerr)
		}
		sq.Timestamp, sq.StartOrEnd = ts, soe
	}
	return sq, nil
}

func buildChild(spec map[string]any, key string) (parser.Expr, error) {
	v, ok := spec[key]
	if !ok {
		return nil, fmt.Errorf("missing %q", key)
	}
	e, err := build(v)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return e, nil
}

func buildArgs(v any) (parser.Expressions, error) {
	if v == nil {
		return nil, nil
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("call args must be a list, got %T", v)
	}
	out := make(parser.Expressions, len(list))
	for i, item := range list {
		e, err := build(item)
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		out[i] = e
	}
	return out, nil
}

// matcherTypes maps the matcher operator tokens to labels.MatchType.
var matcherTypes = map[string]labels.MatchType{
	"=":  labels.MatchEqual,
	"!=": labels.MatchNotEqual,
	"=~": labels.MatchRegexp,
	"!~": labels.MatchNotRegexp,
}

func buildMatchers(v any) ([]*labels.Matcher, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("matchers must be a list, got %T", v)
	}
	out := make([]*labels.Matcher, 0, len(list))
	for i, item := range list {
		spec, err := objectSpec(item, "name", "type", "value")
		if err != nil {
			return nil, fmt.Errorf("matcher %d: %w", i, err)
		}
		name, _ := spec["name"].(string)
		typ, _ := spec["type"].(string)
		value, _ := spec["value"].(string)
		if name == "" {
			return nil, fmt.Errorf("matcher %d requires a name", i)
		}
		mt, ok := matcherTypes[typ]
		if !ok {
			return nil, fmt.Errorf("matcher %d: unknown type %q (want =, !=, =~, or !~)", i, typ)
		}
		m, err := labels.NewMatcher(mt, name, value)
		if err != nil {
			return nil, fmt.Errorf("matcher %d: %w", i, err)
		}
		out = append(out, m)
	}
	return out, nil
}

// binaryOps maps operator tokens to their parser ItemType.
var binaryOps = map[string]parser.ItemType{
	"+": parser.ADD, "-": parser.SUB, "*": parser.MUL, "/": parser.DIV, "%": parser.MOD, "^": parser.POW, "atan2": parser.ATAN2,
	"==": parser.EQLC, "!=": parser.NEQ, "<": parser.LSS, "<=": parser.LTE, ">": parser.GTR, ">=": parser.GTE,
	"and": parser.LAND, "or": parser.LOR, "unless": parser.LUNLESS,
}

// aggOps maps aggregation operator names to their parser ItemType.
var aggOps = map[string]parser.ItemType{
	"sum": parser.SUM, "avg": parser.AVG, "min": parser.MIN, "max": parser.MAX,
	"count": parser.COUNT, "count_values": parser.COUNT_VALUES, "quantile": parser.QUANTILE,
	"stddev": parser.STDDEV, "stdvar": parser.STDVAR, "topk": parser.TOPK, "bottomk": parser.BOTTOMK,
	"group": parser.GROUP,
}

// duration parses a PromQL duration string (e.g. "5m", "1w") into a time.Duration. A leading "-" is accepted for a negative offset; model.ParseDuration itself rejects negatives, so the sign is handled here.
func duration(v any) (time.Duration, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("duration must be a string like \"5m\", got %T", v)
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	d, err := model.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	if neg {
		return -time.Duration(d), nil
	}
	return time.Duration(d), nil
}

// parseAt parses an `@` modifier into the timestamp and start/end fields shared by vector selectors and subqueries: a number is a Unix timestamp in seconds, or the strings "start"/"end". Exactly one of the two returns is set.
func parseAt(at any) (*int64, parser.ItemType, error) {
	switch a := at.(type) {
	case string:
		switch a {
		case "start":
			return nil, parser.START, nil
		case "end":
			return nil, parser.END, nil
		default:
			return nil, 0, fmt.Errorf("at must be a number of seconds or \"start\"/\"end\", got %q", a)
		}
	default:
		f, err := toFloat(at)
		if err != nil {
			return nil, 0, fmt.Errorf("at: %w", err)
		}
		// Reject values that cannot be a real timestamp in milliseconds; int64(f*1000) on an out-of-range float is implementation-defined and would silently produce a bogus timestamp. 1e15 seconds is already past the year 33000, well beyond any plausible @ modifier.
		if math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > 1e15 {
			return nil, 0, fmt.Errorf("at timestamp %v is out of range", f)
		}
		// Round rather than truncate, matching the parser's timestamp.FromFloatSeconds.
		ms := int64(math.Round(f * 1000))
		return &ms, 0, nil
	}
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float64:
		return n, nil
	default:
		if s, ok := v.(fmt.Stringer); ok {
			return strconv.ParseFloat(s.String(), 64)
		}
		return 0, fmt.Errorf("expected a number, got %T", v)
	}
}

func stringList(v any) ([]string, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list of strings, got %T", v)
	}
	out := make([]string, 0, len(list))
	for i, item := range list {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("element %d must be a string, got %T", i, item)
		}
		out = append(out, s)
	}
	return out, nil
}

func objectSpec(v any, allowed ...string) (map[string]any, error) {
	spec, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected an object, got %T", v)
	}
	for k := range spec {
		if !contains(allowed, k) {
			return nil, fmt.Errorf("unknown key %q", k)
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
