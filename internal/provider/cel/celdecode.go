package cel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*CELDecodeFunction)(nil)

type CELDecodeFunction struct{}

func NewCELDecodeFunction() function.Function { return &CELDecodeFunction{} }

func (f *CELDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celdecode"
}

func (f *CELDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a CEL expression string into a celencode data tree",
		MarkdownDescription: "Parses a [CEL](https://cel.dev) expression string and returns it as the HCL data tree that `celencode` consumes, so `provider::burnham::celencode(provider::burnham::celdecode(expr))` round-trips to the canonical form of `expr`. Primarily a tool for testing and for migrating hand-written CEL into the data model.\n\nThe optional second argument selects the notation returned:\n\n- `canonical`: the verbose `cel/expr/syntax.proto` field-name form (`call_expr`, `ident_expr`, `select_expr`, `const_expr`, `list_expr`, `struct_expr`); operators are calls with the canonical function names.\n- `standard` (default): the readable form: type-name keys (`ident`, `call`, ...), folded `ident` reference paths, bare literals, and CEL operator tokens (`\"==\"`, `\"&&\"`, `\"in\"`). Nested `&&`/`||` are flattened into a single variadic list.\n- `aliased`: like `standard` but with the friendly word aliases (`and`, `or`, `not`, `eq`, `ne`, `lt`, ...).\n\nAll three re-encode through `celencode` to the same CEL string. Validation is syntax-only (via cel-go with optional types and two-variable comprehensions enabled), so any syntactically valid CEL decodes. The return is a dynamic value; CEL list literals decode to Terraform tuples (heterogeneous), which `celencode` accepts on the way back.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "A CEL expression string.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. Key: `notation`, one of `canonical`, `standard` (default), or `aliased`.",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.DynamicReturn{},
	}
}

func (f *CELDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &optsArgs))
	if resp.Error != nil {
		return
	}

	if len(expr) > celMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", celMaxInputBytes))
		return
	}
	if optionsHaveUnknown(optsArgs) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicUnknown()))
		return
	}

	notation, ferr := decodeOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	node, err := Decode(expr, notation)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	value, err := nodeToAttr(node)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}

// decodeOptions parses the optional { notation } object.
// The options object is parameter index 1.
func decodeOptions(opts []types.Dynamic) (string, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ notation = "standard" }`)
	if ferr != nil {
		return "", ferr
	}
	notation := NotationStandard
	for k, v := range attrs {
		switch k {
		case "notation":
			s, ok := v.(basetypes.StringValue)
			if !ok || s.IsNull() || s.IsUnknown() {
				return "", function.NewArgumentFuncError(1, "options.notation must be a string")
			}
			switch s.ValueString() {
			case NotationCanonical, NotationStandard, NotationAliased:
				notation = s.ValueString()
			default:
				return "", function.NewArgumentFuncError(1, fmt.Sprintf("options.notation must be one of canonical, standard, or aliased; got %q", s.ValueString()))
			}
		default:
			return "", function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; the only supported key is notation", k))
		}
	}
	return notation, nil
}
