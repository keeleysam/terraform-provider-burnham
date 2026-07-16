package dataformat

import (
	_ "embed"

	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/andygrunwald/vdf"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ function.Function = (*VDFDecodeFunction)(nil)

//go:embed descriptions/vdfdecode.md
var vdfdecodeDescription string

type VDFDecodeFunction struct{}

func NewVDFDecodeFunction() function.Function {
	return &VDFDecodeFunction{}
}

func (f *VDFDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "vdfdecode"
}

func (f *VDFDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a Valve Data Format (VDF) string into a Terraform value",
		MarkdownDescription: vdfdecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A VDF string to parse.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *VDFDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}

	parser := vdf.NewParser(strings.NewReader(input))
	goVal, err := parser.Parse()
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to parse VDF: "+err.Error())
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert VDF value: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*VDFEncodeFunction)(nil)

//go:embed descriptions/vdfencode.md
var vdfencodeDescription string

type VDFEncodeFunction struct{}

func NewVDFEncodeFunction() function.Function {
	return &VDFEncodeFunction{}
}

func (f *VDFEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "vdfencode"
}

func (f *VDFEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a value as a Valve Data Format (VDF) string",
		MarkdownDescription: vdfencodeDescription,
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
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), nil) {
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
			b.WriteString(fmt.Sprintf("%s%s\t\t%s\n", indent, vdfQuote(key), vdfQuote(v.ValueString())))

		case basetypes.ObjectValue:
			b.WriteString(fmt.Sprintf("%s%s\n%s{\n", indent, vdfQuote(key), indent))
			if err := writeVDFObject(b, v.Attributes(), depth+1); err != nil {
				return err
			}
			b.WriteString(fmt.Sprintf("%s}\n", indent))

		case basetypes.NumberValue:
			f := v.ValueBigFloat()
			b.WriteString(fmt.Sprintf("%s%s\t\t%s\n", indent, vdfQuote(key), vdfQuote(f.Text('f', -1))))

		case basetypes.BoolValue:
			s := "0"
			if v.ValueBool() {
				s = "1"
			}
			b.WriteString(fmt.Sprintf("%s%s\t\t%s\n", indent, vdfQuote(key), vdfQuote(s)))

		default:
			return fmt.Errorf("key %q: unsupported type %T (VDF only supports strings and nested objects)", key, val)
		}
	}

	return nil
}

/*
vdfQuote wraps s in double quotes using VDF-native escaping rather than Go's %q.

The andygrunwald/vdf parser only interprets \" and \\ inside a quoted string; any other backslash sequence has its backslash dropped (so Go's %q output of "\n" and "\t" would decode back as the letters "n" and "t"). Newlines and tabs, on the other hand, round-trip fine when written literally because the parser scans them as ordinary whitespace/line-ending runes inside the quotes. So we escape only the backslash and the quotation mark and emit every other rune, including control characters, verbatim.
*/
func vdfQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
