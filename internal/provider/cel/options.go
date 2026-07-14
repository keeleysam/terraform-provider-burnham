package cel

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// celFormatOptions parses the optional options object shared by celencode and celvalidate and returns output-formatting options.
// The only key is `format`, an object mirroring cel-go's unparser options plus celfmt's pretty-printing; each sub-key is optional and left at the backend default when omitted.
// The options object is at parameter index 1.
//
//	{ format = { pretty = true, indent = "  ", wrap_on_column = 60, wrap_on_operators = ["&&", "||"], wrap_after_column_limit = true, always_comma = false } }
func celFormatOptions(opts []types.Dynamic) ([]FormatOption, *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ format = { pretty = true, indent = "  " } }`)
	if ferr != nil {
		return nil, ferr
	}
	var out []FormatOption
	for k, v := range attrs {
		switch k {
		case "format":
			obj, ok := v.(basetypes.ObjectValue)
			if !ok || obj.IsNull() || obj.IsUnknown() {
				return nil, function.NewArgumentFuncError(1, "options.format must be an object")
			}
			fo, ferr := formatFieldOptions(obj.Attributes())
			if ferr != nil {
				return nil, ferr
			}
			out = append(out, fo...)
		default:
			return nil, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; the only supported key is format", k))
		}
	}
	return out, nil
}

func formatFieldOptions(attrs map[string]attr.Value) ([]FormatOption, *function.FuncError) {
	var out []FormatOption
	for k, v := range attrs {
		switch k {
		case "wrap_on_column":
			n, err := optionsutil.NumberAttrToInt(v)
			if err != nil {
				return nil, function.NewArgumentFuncError(1, "options.format.wrap_on_column: "+err.Error())
			}
			if n <= 0 {
				return nil, function.NewArgumentFuncError(1, "options.format.wrap_on_column must be a positive integer")
			}
			out = append(out, WrapColumn(n))
		case "wrap_on_operators":
			var elems []attr.Value
			switch lv := v.(type) {
			case basetypes.ListValue:
				elems = lv.Elements()
			case basetypes.TupleValue:
				elems = lv.Elements()
			default:
				return nil, function.NewArgumentFuncError(1, "options.format.wrap_on_operators must be a list of operator strings")
			}
			syms := make([]string, 0, len(elems))
			for _, el := range elems {
				s, ok := el.(basetypes.StringValue)
				if !ok || s.IsNull() || s.IsUnknown() {
					return nil, function.NewArgumentFuncError(1, "options.format.wrap_on_operators entries must be strings")
				}
				syms = append(syms, s.ValueString())
			}
			// An empty list means "use the default" (&& and ||), not "never wrap".
			if len(syms) > 0 {
				out = append(out, WrapOperators(syms...))
			}
		case "wrap_after_column_limit":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() || b.IsUnknown() {
				return nil, function.NewArgumentFuncError(1, "options.format.wrap_after_column_limit must be a boolean")
			}
			out = append(out, WrapAfter(b.ValueBool()))
		case "pretty":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() || b.IsUnknown() {
				return nil, function.NewArgumentFuncError(1, "options.format.pretty must be a boolean")
			}
			if b.ValueBool() {
				out = append(out, Pretty())
			}
		case "indent":
			s, ok := v.(basetypes.StringValue)
			if !ok || s.IsNull() || s.IsUnknown() {
				return nil, function.NewArgumentFuncError(1, "options.format.indent must be a string")
			}
			out = append(out, Indent(s.ValueString()))
		case "always_comma":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() || b.IsUnknown() {
				return nil, function.NewArgumentFuncError(1, "options.format.always_comma must be a boolean")
			}
			if b.ValueBool() {
				out = append(out, AlwaysComma())
			}
		default:
			return nil, function.NewArgumentFuncError(1, fmt.Sprintf("unknown format option %q; supported: wrap_on_column, wrap_on_operators, wrap_after_column_limit, pretty, indent, always_comma", k))
		}
	}
	return out, nil
}
