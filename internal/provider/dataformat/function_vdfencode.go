package dataformat

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ function.Function = (*VDFEncodeFunction)(nil)

type VDFEncodeFunction struct{}

func NewVDFEncodeFunction() function.Function {
	return &VDFEncodeFunction{}
}

func (f *VDFEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "vdfencode"
}

func (f *VDFEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as a Valve Data Format (VDF) string",
		Description: "Encodes a Terraform object as a VDF string (used by Steam/Source engine). " +
			"VDF is a nested key-value format — all values must be strings or nested objects.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "An object to encode as VDF. Values must be strings or nested objects.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *VDFEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	obj, ok := value.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must be an object."))
		return
	}

	var b strings.Builder
	if err := writeVDFObject(&b, obj.Attributes(), 0); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode VDF: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, b.String()))
}

// writeVDFObject writes a VDF object (map of key-value pairs) at the given indentation depth.
func writeVDFObject(b *strings.Builder, attrs map[string]attr.Value, depth int) error {
	indent := strings.Repeat("\t", depth)

	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := attrs[key]

		switch v := val.(type) {
		case basetypes.StringValue:
			b.WriteString(fmt.Sprintf("%s%q\t\t%q\n", indent, key, v.ValueString()))

		case basetypes.ObjectValue:
			b.WriteString(fmt.Sprintf("%s%q\n%s{\n", indent, key, indent))
			if err := writeVDFObject(b, v.Attributes(), depth+1); err != nil {
				return err
			}
			b.WriteString(fmt.Sprintf("%s}\n", indent))

		case basetypes.NumberValue:
			f := v.ValueBigFloat()
			b.WriteString(fmt.Sprintf("%s%q\t\t%q\n", indent, key, f.Text('f', -1)))

		case basetypes.BoolValue:
			s := "0"
			if v.ValueBool() {
				s = "1"
			}
			b.WriteString(fmt.Sprintf("%s%q\t\t%q\n", indent, key, s))

		default:
			return fmt.Errorf("key %q: unsupported type %T (VDF only supports strings and nested objects)", key, val)
		}
	}

	return nil
}
