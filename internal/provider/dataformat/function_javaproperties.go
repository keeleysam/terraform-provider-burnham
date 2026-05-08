package dataformat

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/magiconair/properties"
)

var _ function.Function = (*JavaPropertiesDecodeFunction)(nil)

type JavaPropertiesDecodeFunction struct{}

func NewJavaPropertiesDecodeFunction() function.Function { return &JavaPropertiesDecodeFunction{} }

func (f *JavaPropertiesDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "javapropertiesdecode"
}

func (f *JavaPropertiesDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a Java .properties file into a string-keyed object",
		MarkdownDescription: "Parses a Java [`.properties`](https://en.wikipedia.org/wiki/.properties) file body into an object. Comments (`#` and `!`), `=`/`:`/whitespace separators, line continuation via trailing `\\`, and `\\uXXXX` Unicode escapes are all handled per the standard `java.util.Properties` semantics.\n\nBy default property expansion (`${other.key}` substitution) is disabled — values are returned exactly as written. All values are returned as strings.\n\nBacked by [magiconair/properties](https://github.com/magiconair/properties), an actively-maintained Go implementation.\n\n**Common uses:** ingesting Spring/Quarkus `application.properties`, JBoss/WildFly server config, or any JVM-shop runtime configuration.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A Java .properties string.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JavaPropertiesDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	loader := &properties.Loader{Encoding: properties.UTF8, DisableExpansion: true}
	props, err := loader.LoadBytes([]byte(input))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to parse .properties: "+err.Error())
		return
	}

	keys := props.Keys()
	attrTypes := make(map[string]attr.Type, len(keys))
	attrValues := make(map[string]attr.Value, len(keys))
	for _, k := range keys {
		v, _ := props.Get(k)
		attrTypes[k] = types.StringType
		attrValues[k] = types.StringValue(v)
	}

	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to build object: "+diags.Errors()[0].Detail()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

var _ function.Function = (*JavaPropertiesEncodeFunction)(nil)

type JavaPropertiesEncodeFunction struct{}

func NewJavaPropertiesEncodeFunction() function.Function { return &JavaPropertiesEncodeFunction{} }

func (f *JavaPropertiesEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "javapropertiesencode"
}

func (f *JavaPropertiesEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a string-keyed object as a Java .properties file body",
		MarkdownDescription: "Encodes a flat string-keyed object as `key=value` lines in alphabetical key order, ready to write to disk. Numeric and boolean values are stringified. Keys and values are escaped according to `java.util.Properties` rules: leading whitespace, `=`, `:`, `#`, `!`, `\\`, and control characters are backslash-escaped; non-ASCII characters are emitted as `\\uXXXX` escapes for portability across legacy ISO-8859-1 readers.\n\nOutput is hand-formatted rather than written via the magiconair/properties library writer, so that keys are sorted (the library preserves insertion order) and the output has no leading metadata block. Nested objects and lists are not allowed.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "A flat object whose attributes are the property keys; values must be primitives.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *JavaPropertiesEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	entries, err := javaPropertiesFromValue(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, renderJavaProperties(entries)))
}

func javaPropertiesFromValue(v attr.Value) (map[string]string, error) {
	switch tv := v.(type) {
	case basetypes.ObjectValue:
		out := make(map[string]string, len(tv.Attributes()))
		for k, av := range tv.Attributes() {
			s, err := dotenvScalarString(av)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			out[k] = s
		}
		return out, nil
	case basetypes.MapValue:
		out := make(map[string]string, len(tv.Elements()))
		for k, av := range tv.Elements() {
			s, err := dotenvScalarString(av)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			out[k] = s
		}
		return out, nil
	default:
		return nil, fmt.Errorf("propertiesencode requires an object or map, got %T", v)
	}
}

// renderJavaProperties hand-rolls the .properties output instead of using magiconair/properties' Properties.Write(). The library writer doesn't sort keys (we want stable output), emits blank-line padding around comments we don't have, and doesn't emit `\uXXXX` escapes for non-ASCII keys/values that we want for portability with legacy ISO-8859-1 readers.
func renderJavaProperties(entries map[string]string) string {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteString(escapeJavaPropertiesKey(k))
		buf.WriteByte('=')
		buf.WriteString(escapeJavaPropertiesValue(entries[k]))
		buf.WriteByte('\n')
	}
	return buf.String()
}

func escapeJavaPropertiesKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		switch r {
		case ' ', '\t':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '=', ':', '#', '!':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r > 0x7E || (r < 0x20 && r != '\t') {
				fmt.Fprintf(&b, "\\u%04X", r)
			} else {
				b.WriteRune(r)
			}
		}
		_ = i
	}
	return b.String()
}

func escapeJavaPropertiesValue(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	leading := true
	for _, r := range s {
		switch r {
		case ' ':
			if leading {
				b.WriteString(`\ `)
			} else {
				b.WriteRune(r)
			}
			continue
		case '\t':
			b.WriteString(`\t`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r > 0x7E || (r < 0x20 && r != '\t') {
				fmt.Fprintf(&b, "\\u%04X", r)
			} else {
				b.WriteRune(r)
			}
		}
		leading = false
	}
	return b.String()
}
