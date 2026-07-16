package dataformat

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/ini.v1"
)

var _ function.Function = (*INIDecodeFunction)(nil)

type INIDecodeFunction struct{}

func NewINIDecodeFunction() function.Function {
	return &INIDecodeFunction{}
}

func (f *INIDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "inidecode"
}

func (f *INIDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse an INI file into a Terraform value",
		MarkdownDescription: "Parses an INI string into a Terraform object. The result is a two-level map: section names at the outer level, key/value pairs at the inner level. Keys outside any section (\"global\" keys) are placed under the empty-string key (`\"\"`) so the structure stays uniform.\n\nAll values are strings, since INI has no native type system. Convert numerically/booleanly as needed in HCL.\n\n**Common uses:** reading legacy application config (`my.cnf`, `php.ini`, `.gitconfig`-style files), normalizing operator-edited config into a typed Terraform value, or feeding INI content into a `templatefile` substitution.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "An INI string to parse.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *INIDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowBooleanKeys:        true,
		SkipUnrecognizableLines: false,
		// Read double-quoted values verbatim so values that renderINI quoted
		// (surrounding whitespace, inline ; or #) survive the round trip.
		UnescapeValueDoubleQuotes: true,
	}, []byte(input))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to parse INI: "+err.Error())
		return
	}

	sections := cfg.Sections()
	sectionTypes := make(map[string]attr.Type, len(sections))
	sectionValues := make(map[string]attr.Value, len(sections))

	for _, section := range sections {
		name := section.Name()
		// The ini library uses "DEFAULT" for the global section; we use "".
		if name == "DEFAULT" {
			name = ""
		}

		keys := section.Keys()
		keyTypes := make(map[string]attr.Type, len(keys))
		keyValues := make(map[string]attr.Value, len(keys))

		for _, key := range keys {
			keyTypes[key.Name()] = types.StringType
			keyValues[key.Name()] = types.StringValue(key.String())
		}

		sectionObj := types.ObjectValueMust(keyTypes, keyValues)
		sectionTypes[name] = sectionObj.Type(ctx)
		sectionValues[name] = sectionObj
	}

	result, diags := types.ObjectValue(sectionTypes, sectionValues)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to build result: "+diags.Errors()[0].Detail()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(result)))
}

// iniSectionNames returns sorted section names from a Terraform object,
// with "" (global) first if present.
func iniSectionNames(attrs map[string]attr.Value) []string {
	names := make([]string, 0, len(attrs))
	for k := range attrs {
		names = append(names, k)
	}
	sort.Strings(names)
	// Move "" to front if present.
	for i, n := range names {
		if n == "" {
			names = append(names[:i], names[i+1:]...)
			names = append([]string{""}, names...)
			break
		}
	}
	return names
}

// iniKeyNames returns sorted key names from a section's attributes.
func iniKeyNames(attrs map[string]attr.Value) []string {
	names := make([]string, 0, len(attrs))
	for k := range attrs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// iniValueNeedsQuoting reports whether v would be altered by the parser if
// written bare, and so must be wrapped in double quotes to survive a round trip.
func iniValueNeedsQuoting(v string) bool {
	if v == "" {
		return false
	}
	// Leading or trailing whitespace is trimmed by the parser.
	if strings.TrimSpace(v) != v {
		return true
	}
	// ; and # start inline comments and would truncate the value.
	if strings.ContainsAny(v, ";#") {
		return true
	}
	// A double quote (in particular a leading one) triggers quoted-value parsing.
	if strings.Contains(v, "\"") {
		return true
	}
	return false
}

// iniQuoteValue renders v for the right-hand side of a key/value pair, quoting
// and escaping it when a bare value would not round-trip through the parser.
func iniQuoteValue(v string) string {
	if !iniValueNeedsQuoting(v) {
		return v
	}
	return "\"" + strings.ReplaceAll(v, "\"", "\\\"") + "\""
}

// renderINI builds an INI string from a Terraform object structure. It returns
// an error when a value contains a newline, which INI has no standard form for.
func renderINI(sections map[string]attr.Value) (string, error) {
	var b strings.Builder

	sectionNames := iniSectionNames(sections)

	first := true
	for _, sectionName := range sectionNames {
		sectionVal := sections[sectionName]

		var keys map[string]attr.Value
		switch v := sectionVal.(type) {
		case types.Object:
			keys = v.Attributes()
		default:
			continue
		}

		if len(keys) == 0 && sectionName == "" {
			continue
		}

		if !first && sectionName != "" {
			b.WriteString("\n")
		}

		if sectionName != "" {
			b.WriteString("[" + sectionName + "]\n")
		}

		keyNames := iniKeyNames(keys)
		for _, keyName := range keyNames {
			val := keys[keyName]
			var strVal string
			switch v := val.(type) {
			case types.String:
				strVal = v.ValueString()
			case types.Number:
				f := v.ValueBigFloat()
				strVal = f.Text('f', -1)
			case types.Bool:
				if v.ValueBool() {
					strVal = "true"
				} else {
					strVal = "false"
				}
			default:
				strVal = ""
			}
			if strings.ContainsAny(strVal, "\n\r") {
				return "", fmt.Errorf("value for key %q in section %q contains a newline, which INI cannot represent", keyName, sectionName)
			}
			b.WriteString(keyName + " = " + iniQuoteValue(strVal) + "\n")
		}

		first = false
	}

	return b.String(), nil
}

var _ function.Function = (*INIEncodeFunction)(nil)

type INIEncodeFunction struct{}

func NewINIEncodeFunction() function.Function {
	return &INIEncodeFunction{}
}

func (f *INIEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "iniencode"
}

func (f *INIEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a value as an INI file",
		MarkdownDescription: "Encodes a Terraform object as an INI string. The input must be a two-level map: section names at the outer level, key/value pairs at the inner level. Keys under the empty-string key (`\"\"`) are rendered as global keys before any `[section]` header.\n\nAll values are converted to strings; sections and keys are written in alphabetical order for deterministic output.\n\n**Common uses:** generating legacy application config files via `local_file`, or rendering INI snippets to be assembled into a larger config through `templatefile`.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "An object of {section_name = {key = value}} to encode as INI.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *INIEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), nil) {
		return
	}

	obj, ok := value.UnderlyingValue().(types.Object)
	if !ok {
		resp.Error = function.NewArgumentFuncError(0, "value must be an object with section names as keys")
		return
	}

	result, err := renderINI(obj.Attributes())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}
