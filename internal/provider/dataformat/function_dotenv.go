package dataformat

import (
	_ "embed"

	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/joho/godotenv"
)

var _ function.Function = (*DotenvDecodeFunction)(nil)

//go:embed descriptions/dotenvdecode.md
var dotenvdecodeDescription string

type DotenvDecodeFunction struct{}

func NewDotenvDecodeFunction() function.Function { return &DotenvDecodeFunction{} }

func (f *DotenvDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dotenvdecode"
}

func (f *DotenvDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a dotenv (.env) file into a string-to-string map",
		MarkdownDescription: dotenvdecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A dotenv-formatted string.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *DotenvDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	parsed, err := godotenv.Unmarshal(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to parse dotenv: "+err.Error())
		return
	}

	asMap := make(map[string]interface{}, len(parsed))
	for k, v := range parsed {
		asMap[k] = v
	}

	tfVal, err := goMapToObject(asMap)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*DotenvEncodeFunction)(nil)

//go:embed descriptions/dotenvencode.md
var dotenvencodeDescription string

type DotenvEncodeFunction struct{}

func NewDotenvEncodeFunction() function.Function { return &DotenvEncodeFunction{} }

func (f *DotenvEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dotenvencode"
}

func (f *DotenvEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a string-keyed object as a dotenv (.env) file body",
		MarkdownDescription: dotenvencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "An object whose attributes are the dotenv keys; values must be primitives (string, number, bool).",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *DotenvEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
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
		mv, isMap := value.UnderlyingValue().(basetypes.MapValue)
		if !isMap {
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("dotenvencode requires an object or map, got %T", value.UnderlyingValue()))
			return
		}
		entries, err := dotenvFromMapValue(mv)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		rendered, err := renderDotenv(entries)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, rendered))
		return
	}

	entries, err := dotenvFromObjectValue(obj)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	rendered, err := renderDotenv(entries)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, rendered))
}

func dotenvFromObjectValue(obj basetypes.ObjectValue) (map[string]string, error) {
	out := make(map[string]string, len(obj.Attributes()))
	for k, v := range obj.Attributes() {
		s, err := dotenvScalarString(v)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		out[k] = s
	}
	return out, nil
}

func dotenvFromMapValue(mv basetypes.MapValue) (map[string]string, error) {
	out := make(map[string]string, len(mv.Elements()))
	for k, v := range mv.Elements() {
		s, err := dotenvScalarString(v)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		out[k] = s
	}
	return out, nil
}

func dotenvScalarString(v interface{}) (string, error) {
	switch sv := v.(type) {
	case basetypes.StringValue:
		return sv.ValueString(), nil
	case basetypes.BoolValue:
		if sv.ValueBool() {
			return "true", nil
		}
		return "false", nil
	case basetypes.NumberValue:
		f := sv.ValueBigFloat()
		if f.IsInt() {
			i, _ := f.Int(nil)
			return i.String(), nil
		}
		return f.Text('g', -1), nil
	default:
		return "", fmt.Errorf("dotenv values must be string, number, or bool — got %T", v)
	}
}

func renderDotenv(entries map[string]string) (string, error) {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		if err := validateDotenvKey(k); err != nil {
			return "", err
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(quoteDotenvValue(entries[k]))
		buf.WriteByte('\n')
	}
	return buf.String(), nil
}

// validateDotenvKey enforces POSIX shell-compatible identifier rules: starts with a letter or underscore, then letters / digits / underscores. Anything else (whitespace, =, quotes, dots, dashes) would produce a malformed .env file that downstream parsers either reject or misinterpret.
func validateDotenvKey(k string) error {
	if k == "" {
		return fmt.Errorf("dotenv key cannot be empty")
	}
	for i, r := range k {
		valid := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' ||
			(i > 0 && r >= '0' && r <= '9')
		if !valid {
			return fmt.Errorf("invalid dotenv key %q: must match [A-Za-z_][A-Za-z0-9_]*", k)
		}
	}
	return nil
}

func quoteDotenvValue(s string) string {
	if s == "" {
		return ""
	}
	needsQuoting := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '"' || r == '\'' || r == '\\' || r == '$' || r == '#' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '$':
			// godotenv interpolates ${VAR}/$VAR in double-quoted values at decode
			// time; a leading backslash (\$) is stripped back to a literal $, so
			// escaping here is what keeps values with $ round-tripping.
			b.WriteString(`\$`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			// godotenv only reverses \n and \r on decode; an emitted \t decodes to
			// a literal "t", so write a real tab byte, which round-trips verbatim.
			b.WriteByte('\t')
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
