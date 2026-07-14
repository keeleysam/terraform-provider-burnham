package cel

import (
	"fmt"

	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
)

type opKind int

const (
	kBinary   opKind = iota // exactly 2 operands
	kUnary                  // exactly 1 operand
	kTernary                // exactly 3 operands
	kVariadic               // 1+ operands, folded left-associatively into nested binary calls
)

type opSpec struct {
	fn   string
	kind opKind
}

// opTable maps both CEL surface tokens and friendly word aliases to the canonical operator function name.
// Tokens and aliases share entries, so an alias is pure sugar with no separate code path.
// The set is closed and disjoint from the node type-name keys and the *_expr field names.
//
// The "-" token is not here: it is unary negate or binary subtract by arity, handled specially in buildOperator.
var opTable = map[string]opSpec{
	"==": {operators.Equals, kBinary}, "eq": {operators.Equals, kBinary},
	"!=": {operators.NotEquals, kBinary}, "ne": {operators.NotEquals, kBinary},
	"<": {operators.Less, kBinary}, "lt": {operators.Less, kBinary},
	"<=": {operators.LessEquals, kBinary}, "le": {operators.LessEquals, kBinary},
	">": {operators.Greater, kBinary}, "gt": {operators.Greater, kBinary},
	">=": {operators.GreaterEquals, kBinary}, "ge": {operators.GreaterEquals, kBinary},
	"&&": {operators.LogicalAnd, kVariadic}, "and": {operators.LogicalAnd, kVariadic},
	"||": {operators.LogicalOr, kVariadic}, "or": {operators.LogicalOr, kVariadic},
	"!": {operators.LogicalNot, kUnary}, "not": {operators.LogicalNot, kUnary},
	"neg": {operators.Negate, kUnary},
	"in":  {operators.In, kBinary},
	"?:":  {operators.Conditional, kTernary}, "cond": {operators.Conditional, kTernary},
	"+": {operators.Add, kBinary}, "add": {operators.Add, kBinary},
	"*": {operators.Multiply, kBinary}, "mul": {operators.Multiply, kBinary},
	"/": {operators.Divide, kBinary}, "div": {operators.Divide, kBinary},
	"%": {operators.Modulo, kBinary}, "mod": {operators.Modulo, kBinary},
	"sub": {operators.Subtract, kBinary},
	"[]":  {operators.Index, kBinary}, "index": {operators.Index, kBinary},
}

func isOperator(key string) bool {
	if key == "-" {
		return true
	}
	_, ok := opTable[key]
	return ok
}

// operandsList normalizes an operator's value: a list is the operand sequence, anything else is a single operand (convenient for unary operators).
func operandsList(val any) []any {
	if list, ok := val.([]any); ok {
		return list
	}
	return []any{val}
}

func (e *encoder) buildOperator(key string, val any) (ast.Expr, error) {
	raw := operandsList(val)
	ops := make([]ast.Expr, len(raw))
	for i, o := range raw {
		built, err := e.encode(o)
		if err != nil {
			return nil, fmt.Errorf("operator %q operand %d: %w", key, i, err)
		}
		ops[i] = built
	}

	if key == "-" {
		switch len(ops) {
		case 1:
			return e.f.NewCall(e.id(), operators.Negate, ops[0]), nil
		case 2:
			return e.f.NewCall(e.id(), operators.Subtract, ops[0], ops[1]), nil
		default:
			return nil, fmt.Errorf(`operator "-" takes 1 (negate) or 2 (subtract) operands, got %d`, len(ops))
		}
	}

	spec := opTable[key]
	switch spec.kind {
	case kUnary:
		if len(ops) != 1 {
			return nil, fmt.Errorf("operator %q takes exactly 1 operand, got %d", key, len(ops))
		}
		return e.f.NewCall(e.id(), spec.fn, ops[0]), nil
	case kBinary:
		if len(ops) != 2 {
			return nil, fmt.Errorf("operator %q takes exactly 2 operands, got %d", key, len(ops))
		}
		return e.f.NewCall(e.id(), spec.fn, ops[0], ops[1]), nil
	case kTernary:
		if len(ops) != 3 {
			return nil, fmt.Errorf("operator %q takes exactly 3 operands, got %d", key, len(ops))
		}
		return e.f.NewCall(e.id(), spec.fn, ops[0], ops[1], ops[2]), nil
	case kVariadic:
		if len(ops) == 0 {
			return nil, fmt.Errorf("operator %q takes at least 1 operand, got 0", key)
		}
		acc := ops[0]
		for _, o := range ops[1:] {
			acc = e.f.NewCall(e.id(), spec.fn, acc, o)
		}
		return acc, nil
	}
	return nil, fmt.Errorf("operator %q has no builder", key)
}
