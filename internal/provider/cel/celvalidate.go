package cel

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*CELValidateFunction)(nil)

//go:embed descriptions/celvalidate.md
var celvalidateDescription string

type CELValidateFunction struct{}

func NewCELValidateFunction() function.Function { return &CELValidateFunction{} }

func (f *CELValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "celvalidate"
}

func (f *CELValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a string is a syntactically valid CEL expression",
		MarkdownDescription: celvalidateDescription,
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
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", celMaxInputBytes))
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
