package cel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*CELValidateFunction)(nil)

type CELValidateFunction struct{}

func NewCELValidateFunction() function.Function { return &CELValidateFunction{} }

func (f *CELValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celvalidate"
}

func (f *CELValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a syntactically valid CEL expression",
		MarkdownDescription: "Returns `true` if `expr` is a syntactically valid [CEL](https://cel.dev) expression, `false` otherwise. Unlike `celformat`, it does not fail the plan on invalid input, so it is suitable for a boolean check (for example in a `precondition`).\n\nValidation is syntax-only: it does not require variables or functions to be declared, and it does not check types or evaluate the expression. By default it accepts the extensions real Kubernetes and GCP dialects use (optional types and two-variable comprehensions). Pass `{ strict = true }` to validate against base CEL instead, which rejects optional-navigation syntax (`?.`, `[?]`, `[?x]`, `{?k: v}`); useful for checking portability to a plain CEL host that has not enabled those extensions. Backed by [cel-go](https://github.com/google/cel-go).",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "A CEL expression string to check.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "options",
			Description:        "An optional object. Key: `strict` (bool, default false); when true, validate against base CEL with no extensions, so optional-navigation and other extension syntax is rejected.",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.BoolReturn{},
	}
}

func (f *CELValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &optsArgs))
	if resp.Error != nil {
		return
	}

	if len(expr) > celMaxInputBytes {
		// Over the size guard, report not-valid rather than failing the plan, keeping the "does not fail the plan" contract absolute (an expression this large is not a real one).
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, false))
		return
	}
	if optionsHaveUnknown(optsArgs) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.BoolUnknown()))
		return
	}

	attrs, ferr := optionsutil.SingleOptionsObject(optsArgs, `{ strict = true }`)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	strict := false
	for k, v := range attrs {
		switch k {
		case "strict":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() || b.IsUnknown() {
				resp.Error = function.NewArgumentFuncError(1, "options.strict must be a boolean")
				return
			}
			strict = b.ValueBool()
		default:
			resp.Error = function.NewArgumentFuncError(1, "unknown option key "+k+"; the only supported key is strict")
			return
		}
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, IsValid(expr, strict)))
}
