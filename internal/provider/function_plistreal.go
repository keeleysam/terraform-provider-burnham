package provider

import (
	"context"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*PlistRealFunction)(nil)

type PlistRealFunction struct{}

func NewPlistRealFunction() function.Function {
	return &PlistRealFunction{}
}

func (f *PlistRealFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistreal"
}

func (f *PlistRealFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Create a plist real (floating-point) value",
		Description: "Returns a tagged object representing a plist <real> value. Use this when you need to force a whole number to encode as <real> instead of <integer> in a plist. Fractional numbers like 3.14 are automatically encoded as <real> without needing this helper, but whole numbers like 2 would otherwise become <integer>. The same tagged object format is returned by plistdecode for whole-number <real> elements, enabling seamless round-trips.",
		Parameters: []function.Parameter{
			function.NumberParameter{
				Name:        "value",
				Description: "The numeric value for the <real> element.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PlistRealFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value *big.Float

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	if value == nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must not be null."))
		return
	}

	f64, _ := value.Float64() // accuracy flag, not error
	str := strconv.FormatFloat(f64, 'f', -1, 64)

	obj, err := makePlistTaggedObject(plistTypeReal, str)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}
