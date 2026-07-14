package cel

import (
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
)

// isReferencePath reports whether expr is a pure reference: a bare identifier, a field selection, or an index step (including the optional `?.` / `[?]` forms), chained over such.
// It rejects presence tests (has()), comparisons, arbitrary calls, literals, and aggregates.
// Used to keep `ident` from silently accepting a whole expression.
func isReferencePath(expr ast.Expr) bool {
	switch expr.Kind() {
	case ast.IdentKind:
		return true
	case ast.SelectKind:
		sel := expr.AsSelect()
		if sel.IsTestOnly() {
			return false
		}
		return isReferencePath(sel.Operand())
	case ast.CallKind:
		call := expr.AsCall()
		if call.IsMemberFunction() {
			return false
		}
		args := call.Args()
		switch call.FunctionName() {
		case operators.Index, operators.OptIndex:
			// base[key] / base[?key]: the base must be a reference; the key may be any expression (still a navigation into base).
			return len(args) == 2 && isReferencePath(args[0])
		case operators.OptSelect:
			// base.?field
			return len(args) == 2 && isReferencePath(args[0])
		}
		return false
	default:
		return false
	}
}
