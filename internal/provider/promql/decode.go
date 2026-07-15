package promql

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// Decode parses a PromQL query and returns it as the node tree that Encode consumes.
//
// It is the inverse of Encode on canonical forms: for any valid query q, Encode(Decode(q)) equals Format(q). The parser normalizes as it reads (label matchers sort, redundant braces drop), so Decode reflects the canonical query, not the original byte layout. Every node is fully structured; Decode never emits a raw fragment.
func Decode(query string) (any, error) {
	e, err := parseExpr(query)
	if err != nil {
		return nil, err
	}
	return nodeToTree(e)
}

func nodeToTree(e parser.Expr) (any, error) {
	switch n := e.(type) {
	case *parser.NumberLiteral:
		return numberNode(n.Val), nil
	case *parser.StringLiteral:
		return n.Val, nil
	case *parser.VectorSelector:
		fields, err := selectorFields(n)
		if err != nil {
			return nil, err
		}
		return map[string]any{"vectorSelector": fields}, nil
	case *parser.MatrixSelector:
		vs, ok := n.VectorSelector.(*parser.VectorSelector)
		if !ok {
			return nil, fmt.Errorf("matrix selector wraps unexpected %T", n.VectorSelector)
		}
		fields, err := selectorFields(vs)
		if err != nil {
			return nil, err
		}
		fields["range"] = durationString(n.Range)
		return map[string]any{"matrixSelector": fields}, nil
	case *parser.Call:
		return decodeCall(n)
	case *parser.AggregateExpr:
		return decodeAggregation(n)
	case *parser.BinaryExpr:
		return decodeBinary(n)
	case *parser.SubqueryExpr:
		return decodeSubquery(n)
	case *parser.ParenExpr:
		inner, err := nodeToTree(n.Expr)
		if err != nil {
			return nil, err
		}
		return map[string]any{"paren": inner}, nil
	case *parser.UnaryExpr:
		inner, err := nodeToTree(n.Expr)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case parser.SUB:
			return map[string]any{"neg": inner}, nil
		case parser.ADD:
			return map[string]any{"pos": inner}, nil
		default:
			return nil, fmt.Errorf("unsupported unary operator %q", n.Op)
		}
	case *parser.StepInvariantExpr:
		// The parser only produces this during preprocessing, which ParseExpr does not run; decode defensively anyway.
		return nodeToTree(n.Expr)
	default:
		return nil, fmt.Errorf("unsupported PromQL node type %T", e)
	}
}

// selectorFields builds the shared selector fields (name, matchers, offset, at), the inverse of vectorSelectorFields. Default/empty fields are omitted, so a bare metric decodes to just { name = ... }.
func selectorFields(vs *parser.VectorSelector) (map[string]any, error) {
	f := map[string]any{}
	if vs.Name != "" {
		f["name"] = vs.Name
	}
	matchers := decodeMatchers(vs)
	if len(matchers) > 0 {
		f["matchers"] = matchers
	}
	if vs.OriginalOffset != 0 {
		f["offset"] = durationString(vs.OriginalOffset)
	}
	if at := decodeAt(vs.Timestamp, vs.StartOrEnd); at != nil {
		f["at"] = at
	}
	return f, nil
}

// decodeMatchers returns the explicit label matchers as a sorted list, dropping the implicit __name__ matcher the parser adds for a bare metric name (so it is not duplicated alongside the name field).
func decodeMatchers(vs *parser.VectorSelector) []any {
	ms := make([]*labels.Matcher, 0, len(vs.LabelMatchers))
	for _, m := range vs.LabelMatchers {
		if vs.Name != "" && m.Name == labels.MetricName && m.Type == labels.MatchEqual && m.Value == vs.Name {
			continue // the implicit name matcher, carried by the name field instead
		}
		ms = append(ms, m)
	}
	sort.Slice(ms, func(i, j int) bool {
		if ms[i].Name != ms[j].Name {
			return ms[i].Name < ms[j].Name
		}
		return ms[i].Value < ms[j].Value
	})
	out := make([]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, map[string]any{"name": m.Name, "type": m.Type.String(), "value": m.Value})
	}
	return out
}

// decodeAt reverses parseAt: the strings "start"/"end", or a Unix timestamp in seconds. It returns nil when no @ modifier is set.
func decodeAt(ts *int64, soe parser.ItemType) any {
	switch soe {
	case parser.START:
		return "start"
	case parser.END:
		return "end"
	}
	if ts != nil {
		return numberNode(float64(*ts) / 1000)
	}
	return nil
}

func decodeCall(n *parser.Call) (any, error) {
	args := make([]any, len(n.Args))
	for i, a := range n.Args {
		t, err := nodeToTree(a)
		if err != nil {
			return nil, fmt.Errorf("call %s arg %d: %w", n.Func.Name, i, err)
		}
		args[i] = t
	}
	return map[string]any{"call": map[string]any{"func": n.Func.Name, "args": args}}, nil
}

func decodeAggregation(n *parser.AggregateExpr) (any, error) {
	op, ok := aggOpNames[n.Op]
	if !ok {
		return nil, fmt.Errorf("unsupported aggregation operator %q", n.Op)
	}
	expr, err := nodeToTree(n.Expr)
	if err != nil {
		return nil, err
	}
	agg := map[string]any{"op": op, "expr": expr}
	if n.Param != nil {
		if agg["param"], err = nodeToTree(n.Param); err != nil {
			return nil, err
		}
	}
	if n.Without {
		agg["without"] = stringsToList(n.Grouping)
	} else if len(n.Grouping) > 0 {
		agg["by"] = stringsToList(n.Grouping)
	}
	return map[string]any{"aggregation": agg}, nil
}

func decodeBinary(n *parser.BinaryExpr) (any, error) {
	op, ok := binaryOpNames[n.Op]
	if !ok {
		return nil, fmt.Errorf("unsupported binary operator %q", n.Op)
	}
	lhs, err := nodeToTree(n.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := nodeToTree(n.RHS)
	if err != nil {
		return nil, err
	}
	bin := map[string]any{"op": op, "lhs": lhs, "rhs": rhs}
	if n.ReturnBool {
		bin["bool"] = true
	}
	if vm := n.VectorMatching; vm != nil {
		if vm.On {
			bin["on"] = stringsToList(vm.MatchingLabels)
		} else if len(vm.MatchingLabels) > 0 {
			bin["ignoring"] = stringsToList(vm.MatchingLabels)
		}
		switch vm.Card {
		case parser.CardManyToOne:
			bin["group_left"] = stringsToList(vm.Include)
		case parser.CardOneToMany:
			bin["group_right"] = stringsToList(vm.Include)
		}
	}
	return map[string]any{"binaryExpr": bin}, nil
}

func decodeSubquery(n *parser.SubqueryExpr) (any, error) {
	expr, err := nodeToTree(n.Expr)
	if err != nil {
		return nil, err
	}
	sq := map[string]any{"expr": expr, "range": durationString(n.Range)}
	if n.Step != 0 {
		sq["step"] = durationString(n.Step)
	}
	if n.OriginalOffset != 0 {
		sq["offset"] = durationString(n.OriginalOffset)
	}
	if at := decodeAt(n.Timestamp, n.StartOrEnd); at != nil {
		sq["at"] = at
	}
	return map[string]any{"subquery": sq}, nil
}

// durationString renders a duration in PromQL's compact unit form ("5m", "1h30m", "-5m"), the inverse of the duration helper.
func durationString(d time.Duration) string {
	return model.Duration(d).String()
}

// numberNode returns an int64 for an integral value (for clean HCL output) and a float64 otherwise. Both are accepted by Encode. Signed zero is kept as a float64 so its sign survives the round-trip (int64 has no negative zero, and the parser prints -0).
func numberNode(f float64) any {
	if f == math.Trunc(f) && !math.IsInf(f, 0) && math.Abs(f) < 1e18 && !(f == 0 && math.Signbit(f)) {
		return int64(f)
	}
	return f
}

func stringsToList(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// aggOpNames and binaryOpNames invert aggOps and binaryOps for decoding.
var aggOpNames = invertOps(aggOps)
var binaryOpNames = invertOps(binaryOps)

func invertOps(m map[string]parser.ItemType) map[parser.ItemType]string {
	out := make(map[parser.ItemType]string, len(m))
	for name, op := range m {
		out[op] = name
	}
	return out
}
