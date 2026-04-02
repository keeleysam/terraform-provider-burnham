package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"howett.net/plist"
)

var _ function.Function = (*PlistEncodeFunction)(nil)

type PlistEncodeFunction struct{}

func NewPlistEncodeFunction() function.Function {
	return &PlistEncodeFunction{}
}

func (f *PlistEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistencode"
}

func (f *PlistEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Encode a value as an Apple property list",
		Description: "Encodes a Terraform value as an Apple property list (plist) string. Default format is XML. Tagged objects from plistdate() and plistdata() are converted to native plist <date> and <data> elements. Numbers with no fractional part become <integer>, otherwise <real>. When format is \"binary\", the output is base64-encoded. Pass an optional options object with a \"format\" key to override the default.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as a plist.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional options object. Supported keys: \"format\" (string) — \"xml\" (default), \"binary\", or \"openstep\". Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

func (f *PlistEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}

	formatStr := "xml"
	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("options must be an object, got %T", optsArgs[0].UnderlyingValue())))
			return
		}
		parsed, err := getStringOption(obj.Attributes(), "format")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if parsed != "" {
			formatStr = parsed
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	plistFormat, err := parsePlistFormat(formatStr)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), true)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	prepared := goValueForPlistEncode(goVal)

	data, err := plist.MarshalIndent(prepared, plistFormat, "\t")
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode plist: "+err.Error()))
		return
	}

	var result string
	if plistFormat == plist.BinaryFormat {
		result = base64.StdEncoding.EncodeToString(data)
	} else {
		result = string(data)
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}

func parsePlistFormat(s string) (int, error) {
	switch strings.ToLower(s) {
	case "xml":
		return plist.XMLFormat, nil
	case "binary":
		return plist.BinaryFormat, nil
	case "openstep", "gnustep":
		return plist.GNUStepFormat, nil
	default:
		return 0, fmt.Errorf("unsupported plist format %q: must be \"xml\", \"binary\", or \"openstep\"", s)
	}
}
