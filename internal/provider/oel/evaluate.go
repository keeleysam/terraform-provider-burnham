package oel

import okta "github.com/keeleysam/okta-expression-parser"

// EvalContext is the data an Okta EL expression is evaluated against: the user
// profile resolved by user.<attr>, the group IDs the user belongs to (for
// isMemberOfGroup / isMemberOfAnyGroup), group metadata keyed by ID (for the
// isMemberOfGroupName family), and strict property access.
type EvalContext struct {
	UserProfile map[string]any
	GroupIDs    []string
	GroupData   map[string]any
	Strict      bool
}

// Evaluate parses and evaluates an Okta EL expression against ctx, returning
// the result (bool, string, int, float64, nil, or a list/map).
//
// Evaluation covers the group-rule subset the backing parser implements:
// literals, comparisons, boolean logic, the ternary and + operators, the
// String/Arrays/Convert/Iso3166Convert/Groups class functions, the bare
// isMemberOf* group builtins, and user.<attr> paths. Constructs outside that
// set (receiver method calls such as user.getInternalProperty(...), the
// Identity Engine method dialect, user.isMemberOf({...}), getGroups,
// projection, indexing, Elvis, and matches) parse but are not evaluated and
// return an error.
func Evaluate(expr string, ctx EvalContext) (any, error) {
	if err := checkNestingDepth(expr); err != nil {
		return nil, err
	}
	p := okta.New(
		okta.WithUserProfile(ctx.UserProfile),
		okta.WithGroupIDs(ctx.GroupIDs),
		okta.WithGroupData(ctx.GroupData),
		okta.WithStrict(ctx.Strict),
	)
	node, err := p.Parse(expr)
	if err != nil {
		return nil, err
	}
	return p.Eval(node)
}
