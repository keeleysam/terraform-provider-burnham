package oel

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*OELEvaluateFunction)(nil)

//go:embed descriptions/oelevaluate.md
var oelevaluateDescription string

type OELEvaluateFunction struct{}

func NewOELEvaluateFunction() function.Function { return &OELEvaluateFunction{} }

func (f *OELEvaluateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "oelevaluate"
}

func (f *OELEvaluateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Evaluate an Okta EL expression against a sample user profile and group memberships",
		MarkdownDescription: oelevaluateDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "expr",
				Description: "An Okta EL expression string to evaluate.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:               "context",
			Description:        "An optional context object with keys: user (object), group_ids (list of strings), groups (object keyed by group ID), and strict (bool).",
			AllowNullValue:     false,
			AllowUnknownValues: false,
		},
		Return: function.DynamicReturn{},
	}
}

func (f *OELEvaluateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var expr string
	var ctxArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &expr, &ctxArgs))
	if resp.Error != nil {
		return
	}

	if len(expr) > oelMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("expression exceeds maximum supported length of %d bytes", oelMaxInputBytes))
		return
	}
	if optionsHaveUnknown(ctxArgs) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicUnknown()))
		return
	}

	evalCtx, ferr := evaluateContext(ctxArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	result, err := Evaluate(expr, evalCtx)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	value, err := nodeToAttr(result)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(value)))
}

// optionsHaveUnknown reports whether the variadic context object holds any unknown value.
func optionsHaveUnknown(opts []types.Dynamic) bool {
	for _, o := range opts {
		if hasUnknown(o) {
			return true
		}
	}
	return false
}

// evaluateContext parses the optional { user, group_ids, groups, strict } context object at parameter index 1.
func evaluateContext(opts []types.Dynamic) (EvalContext, *function.FuncError) {
	var ec EvalContext
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ user = { department = "Sales" } }`)
	if ferr != nil {
		return ec, ferr
	}
	for k, v := range attrs {
		switch k {
		case "user":
			m, ferr := goObject(v, "options.user")
			if ferr != nil {
				return ec, ferr
			}
			ec.UserProfile = m
		case "groups":
			m, ferr := goObject(v, "options.groups")
			if ferr != nil {
				return ec, ferr
			}
			ec.GroupData = m
		case "group_ids":
			ids, ferr := stringList(v)
			if ferr != nil {
				return ec, ferr
			}
			ec.GroupIDs = ids
		case "strict":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() || b.IsUnknown() {
				return ec, function.NewArgumentFuncError(1, "options.strict must be a boolean")
			}
			ec.Strict = b.ValueBool()
		default:
			return ec, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported: user, group_ids, groups, strict", k))
		}
	}
	return ec, nil
}

// goObject converts an options object value into a native map[string]any.
func goObject(v attr.Value, what string) (map[string]any, *function.FuncError) {
	obj, ok := v.(basetypes.ObjectValue)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return nil, function.NewArgumentFuncError(1, what+" must be an object")
	}
	m, err := attributesToGo(obj.Attributes())
	if err != nil {
		return nil, function.NewArgumentFuncError(1, what+": "+err.Error())
	}
	return m, nil
}

// stringList converts an options list/tuple value into a []string.
func stringList(v attr.Value) ([]string, *function.FuncError) {
	var elems []attr.Value
	switch lv := v.(type) {
	case basetypes.ListValue:
		elems = lv.Elements()
	case basetypes.TupleValue:
		elems = lv.Elements()
	default:
		return nil, function.NewArgumentFuncError(1, "options.group_ids must be a list of strings")
	}
	out := make([]string, 0, len(elems))
	for _, el := range elems {
		s, ok := el.(basetypes.StringValue)
		if !ok || s.IsNull() || s.IsUnknown() {
			return nil, function.NewArgumentFuncError(1, "options.group_ids entries must be strings")
		}
		out = append(out, s.ValueString())
	}
	return out, nil
}
