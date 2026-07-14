package cel

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*CELEncodeFunction)(nil)

/*
TODO (deferred): support emitting CEL comments from the data tree, e.g. a per-node `comment` field that renders as `// ...` in the output.

This does not work today and cannot be bolted on with the current backends. cel-go's AST has no comment representation (its `common/ast` exposes none, the parser has no comment-retention option, and the unparser never emits comments), so a comment attached to a node has nowhere to live once we build the cel-go AST. celfmt does not read comments from the AST either; it recovers them only by scraping the original source text by line position, which a built (source-less) expression has nothing for. Our pretty pipeline also reparses a canonical, comment-free seed before celfmt runs, stripping any comment regardless.

Supporting per-node/structural comments therefore requires our own emitter (the deferred custom formatter): walk the tree and write `// ...` inline while laying out the expression, delegating leaf rendering to cel-go. A single whole-expression leading/trailing comment, by contrast, is trivial string concatenation and needs no AST/library support.
*/
type CELEncodeFunction struct{}

func NewCELEncodeFunction() function.Function { return &CELEncodeFunction{} }

func (f *CELEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celencode"
}

func (f *CELEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Build a CEL expression string from an HCL data tree",
		MarkdownDescription: "Builds a [CEL](https://cel.dev) (Common Expression Language) expression string from a structured HCL value, so you assemble expressions from Terraform data (variables, `for` expressions, `merge`, `concat`) with no string templating. The result is a canonical, deterministic CEL string suitable for GCP IAM / Access Context Manager conditions, Workload Identity Federation, Cloud Armor, Kubernetes CEL, and any other CEL sink.\n\nThe input follows the CEL canonical AST (`cel/expr/syntax.proto`), using its node and field names rather than an invented vocabulary. Comprehension macros are authored in their call form (below), not as a raw `comprehension_expr`. Bare integral numbers become CEL `int`; use `{ const = { double_value = ... } }` for a `double` and `{ const = { uint64_value = ... } }` for an unsigned or large (> 2^63-1) value. Two interchangeable, freely mixable notations are accepted:\n\n**Surface (readable):** bare strings/numbers/bools/null are literals; a bare list is a list literal. A reference (variable, field path, or enum) is the only marked leaf: `{ ident = \"device.os_type\" }` (dotted and `['key']` index paths expand automatically). Operators are CEL surface tokens or friendly aliases: `{ \"==\" = [a, b] }` or `{ eq = [a, b] }`; also `and`/`or`/`not`, `ne`/`lt`/`le`/`gt`/`ge`, `in`, `cond`, `add`/`sub`/`mul`/`div`/`mod`, `neg`, `index`. Calls and macros: `{ call = { function = \"startsWith\", target = { ident = \"resource.name\" }, args = [\"prod-\"] } }`; macros are calls whose function is `has`/`all`/`exists`/`exists_one`/`map`/`filter` with the bound variable as an `{ ident = \"g\" }` argument (`has` takes a single field-selection argument). Explicit literals: `{ const = [\"US\", \"CA\"] }` (recursive: lists, maps, and typed constants like `{ const = { double_value = 1 } }`; note a single-key `const` map whose key is a CEL constant kind, e.g. `{ const = { int64_value = 5 } }`, is read as that typed scalar, not a one-entry map, so spell such a map via `struct_expr` map entries or `raw`). Message construction: `{ struct = { message_name = \"T\", fields = { f = 1 } } }`. Escape hatch: `{ raw = \"a.b.exists(x, x > 0)\" }` embeds hand-written CEL.\n\n**Canonical:** the `syntax.proto` field names, where operators are calls with the canonical function names: `{ call_expr = { function = \"_==_\", args = [ { select_expr = { operand = { ident_expr = { name = \"device\" } }, field = \"os_type\" } }, { ident_expr = { name = \"OsType\" } } ] } }`. Also `const_expr`, `list_expr`, `struct_expr`.\n\n**Optional types** (Kubernetes / GCP): optional navigation is written in an `ident` (or `raw`) path, e.g. `{ ident = \"msg.?field\" }` and `{ ident = \"m[?k]\" }`; optional list elements use `{ optional = <expr> }` inside a list (CEL `[?x]`); optional map/struct entries set `optional_entry = true` on a `struct_expr` entry (CEL `{?k: v}`); and `optional.of`/`orValue`/`hasValue`/`value`/`optMap` are ordinary calls. **Two-variable comprehensions** (`m.all(k, v, ...)`, `transformList`/`transformMap`/`transformMapEntry`) are written as calls with two bound-variable `ident` arguments. `ident` accepts only reference paths (identifier, field, index, optional navigation); a full expression must use `raw`.\n\nThe output is always validated (syntax only, dialect-neutral) before it is returned, so `celencode` can never produce a syntactically invalid CEL string. Backed by [cel-go](https://github.com/google/cel-go), which handles operator precedence, parenthesization, and canonical quoting; the normalized output is stable across runs so it does not churn the plan. Pass a `format` options object to pretty-print or wrap the output (see the options argument).",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "expr",
				Description: "The expression tree, in the surface or canonical notation (mixable).",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. The only key is `format`, an object mirroring cel-go's unparser options plus celfmt's pretty-printing (each optional; omitted keys use the backend default): `pretty` (bool) indents structural containers (lists, maps, call/macro arguments, parenthesized groups) onto multiple lines; `indent` (string, default a tab) is the indent unit; `always_comma` (bool) adds trailing commas; `wrap_on_column` (number) sets the width used to introduce line breaks (default 80 when pretty); `wrap_on_operators` (list, default `[\"&&\", \"||\"]`) and `wrap_after_column_limit` (bool, default true) control operator wrapping. Example: `{ format = { pretty = true, indent = \"  \" } }`. With no `format`, the output is a single canonical line. Note: boolean operator chains are width-wrapped, not one-per-line.",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.StringReturn{},
	}
}

func (f *CELEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &optsArgs))
	if resp.Error != nil {
		return
	}

	if hasUnknown(expr) || optionsHaveUnknown(optsArgs) {
		// A value in the expression or options is unknown at plan time; return an unknown result so the plan proceeds and the value resolves at apply.
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}

	formatOpts, ferr := celFormatOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	node, err := terraformToNode(expr.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to read expression: "+err.Error())
		return
	}

	out, err := Encode(node, formatOpts...)
	if errors.Is(err, errInvalidOutput) {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
