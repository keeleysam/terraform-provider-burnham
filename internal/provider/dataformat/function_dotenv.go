package dataformat

import (
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

type DotenvDecodeFunction struct{}

func NewDotenvDecodeFunction() function.Function { return &DotenvDecodeFunction{} }

func (f *DotenvDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dotenvdecode"
}

func (f *DotenvDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse a dotenv (.env) file into a string-to-string map",
		MarkdownDescription: "Parses a [dotenv](https://github.com/joho/godotenv) (`.env`) file body into an object whose attributes are the file's keys, " +
			"with all values as strings. Comments (`#`) are ignored. Both `KEY=value` and `export KEY=value` lines are accepted. " +
			"Double-quoted values support `\\n`, `\\r`, `\\t` and `${VAR}` interpolation against earlier keys; single-quoted values are taken literally.\n\n" +
			"All values are returned as strings — dotenv has no type system. Cast on the Terraform side with `tonumber()` / `tobool()` if needed.\n\n" +
			"Backed by [joho/godotenv](https://github.com/joho/godotenv), the canonical Go implementation.\n\n" +
			"**Common uses:** ingesting environment files for ECS/Lambda task definitions, container env blocks, or shipping config alongside compiled artifacts.",
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

	parsed, err := godotenv.Unmarshal(input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse dotenv: "+err.Error()))
		return
	}

	asMap := make(map[string]interface{}, len(parsed))
	for k, v := range parsed {
		asMap[k] = v
	}

	tfVal, err := goMapToObject(asMap)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*DotenvEncodeFunction)(nil)

type DotenvEncodeFunction struct{}

func NewDotenvEncodeFunction() function.Function { return &DotenvEncodeFunction{} }

func (f *DotenvEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dotenvencode"
}

func (f *DotenvEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a string-keyed object as a dotenv (.env) file body",
		MarkdownDescription: "Encodes a flat string-keyed object as `KEY=value` lines in alphabetical key order. Numeric and boolean values are stringified. " +
			"Values containing whitespace, quotes, `$`, `\\`, or newline characters are wrapped in double quotes with `\\n`/`\\r`/`\\t`/`\\\"`/`\\\\` escapes — readable round-trip with `dotenvdecode`. " +
			"Nested objects and lists are not allowed.\n\n" +
			"**Common uses:** generating `.env` files for `local_file`, container build contexts, or 12-factor service deployments.",
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

	obj, ok := value.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		mv, isMap := value.UnderlyingValue().(basetypes.MapValue)
		if !isMap {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("dotenvencode requires an object or map, got %T", value.UnderlyingValue())))
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
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
