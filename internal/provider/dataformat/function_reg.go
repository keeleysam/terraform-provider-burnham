package dataformat

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gersonkurz/go-regis3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	regTypeKey      = "__reg_type"
	regValueKey     = "value"
	regTypeDword    = "dword"
	regTypeQword    = "qword"
	regTypeBinary   = "binary"
	regTypeMultiSz  = "multi_sz"
	regTypeExpandSz = "expand_sz"
	regTypeNone     = "none"
	regTypeDelete   = "delete"
)

var _ function.Function = (*RegDecodeFunction)(nil)

type RegDecodeFunction struct{}

func NewRegDecodeFunction() function.Function {
	return &RegDecodeFunction{}
}

func (f *RegDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regdecode"
}

func (f *RegDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse a Windows .reg file into a Terraform value",
		MarkdownDescription: "Decodes a Windows Registry Editor export (.reg) file into a Terraform object. " +
			"Auto-detects Version 4 (REGEDIT4) and Version 5 (Windows Registry Editor Version 5.00). " +
			"The result is a map of registry key paths to maps of value names. " +
			"REG_SZ values become plain strings. Other types use tagged objects with __reg_type and value keys. " +
			"The default value (@) uses the key name \"@\".",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A .reg file string to parse.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *RegDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	root, err := regis3.Parse(input, &regis3.ParseOptions{
		AllowHashtagComments:   true,
		AllowSemicolonComments: true,
		IgnoreWhitespaces:      true,
	})
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse .reg file: "+err.Error()))
		return
	}

	tfVal, err := regKeyTreeToTerraform(root)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert .reg data: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

// regKeyTreeToTerraform walks the parsed key tree and builds a flat map
// of full key paths → value maps.
func regKeyTreeToTerraform(root *regis3.KeyEntry) (attr.Value, error) {
	keyEntries := map[string]*regis3.KeyEntry{}
	collectKeys(root, keyEntries)

	if len(keyEntries) == 0 {
		return types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}), nil
	}

	attrTypes := make(map[string]attr.Type, len(keyEntries))
	attrValues := make(map[string]attr.Value, len(keyEntries))

	paths := make([]string, 0, len(keyEntries))
	for p := range keyEntries {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		entry := keyEntries[path]
		valObj, err := regValuesToTerraform(entry)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", path, err)
		}
		attrTypes[path] = valObj.Type(nil)
		attrValues[path] = valObj
	}

	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		return nil, fmt.Errorf("building result: %s", diags.Errors()[0].Detail())
	}
	return obj, nil
}

// collectKeys recursively walks the key tree, collecting full paths and their key entries.
// Only includes keys that have values or a default value (skips intermediate path nodes).
func collectKeys(key *regis3.KeyEntry, result map[string]*regis3.KeyEntry) {
	path := key.GetPath()
	if path != "" && (len(key.Values()) > 0 || key.DefaultValue() != nil) {
		result[path] = key
	}
	for _, sub := range key.SubKeys() {
		collectKeys(sub, result)
	}
}

// regValuesToTerraform converts a key entry's values (including default) to a Terraform object.
func regValuesToTerraform(key *regis3.KeyEntry) (types.Object, error) {
	values := key.Values()
	attrTypes := make(map[string]attr.Type, len(values)+1)
	attrValues := make(map[string]attr.Value, len(values)+1)

	// Use val.Name() to preserve original casing (map keys are lowercased).
	vals := make([]*regis3.ValueEntry, 0, len(values))
	for _, v := range values {
		vals = append(vals, v)
	}

	// Include the default value if present.
	if def := key.DefaultValue(); def != nil {
		vals = append(vals, def)
	}

	sort.Slice(vals, func(i, j int) bool {
		ni, nj := vals[i].Name(), vals[j].Name()
		if ni == "" {
			ni = "@"
		}
		if nj == "" {
			nj = "@"
		}
		return ni < nj
	})

	for _, val := range vals {
		tfVal, err := regValueToTerraform(val)
		if err != nil {
			return types.Object{}, fmt.Errorf("value %q: %w", val.Name(), err)
		}
		name := val.Name()
		if name == "" {
			name = "@" // Default value
		}
		attrTypes[name] = tfVal.Type(nil)
		attrValues[name] = tfVal
	}

	return types.ObjectValueMust(attrTypes, attrValues), nil
}

// regValueToTerraform converts a single registry value to a Terraform value.
func regValueToTerraform(val *regis3.ValueEntry) (attr.Value, error) {
	if val.RemoveFlag() {
		return makeRegTaggedObject(regTypeDelete, types.StringValue(""))
	}

	switch val.Kind() {
	case regis3.RegNone:
		return makeRegTaggedObject(regTypeNone, types.StringValue(hex.EncodeToString(val.Data())))

	case regis3.RegSz:
		return types.StringValue(val.GetString("")), nil

	case regis3.RegDword:
		return makeRegTaggedObject(regTypeDword, types.StringValue(strconv.FormatUint(uint64(val.GetDword(0)), 10)))

	case regis3.RegQword:
		return makeRegTaggedObject(regTypeQword, types.StringValue(strconv.FormatUint(val.GetQword(0), 10)))

	case regis3.RegBinary:
		return makeRegTaggedObject(regTypeBinary, types.StringValue(hex.EncodeToString(val.Data())))

	case regis3.RegMultiSz:
		strs := val.GetMultiString()
		elems := make([]attr.Value, len(strs))
		elemTypes := make([]attr.Type, len(strs))
		for i, s := range strs {
			elems[i] = types.StringValue(s)
			elemTypes[i] = types.StringType
		}
		list := types.TupleValueMust(elemTypes, elems)
		return makeRegTaggedObject(regTypeMultiSz, list)

	case regis3.RegExpandSz:
		return makeRegTaggedObject(regTypeExpandSz, types.StringValue(val.GetString("")))

	default:
		// Unknown type — encode as binary hex
		return makeRegTaggedObject(regTypeBinary, types.StringValue(hex.EncodeToString(val.Data())))
	}
}

// makeRegTaggedObject creates a tagged object with __reg_type and value (string).
func makeRegTaggedObject(regType string, value attr.Value) (attr.Value, error) {
	attrTypes := map[string]attr.Type{
		regTypeKey:  types.StringType,
		regValueKey: value.Type(nil),
	}
	attrValues := map[string]attr.Value{
		regTypeKey:  types.StringValue(regType),
		regValueKey: value,
	}
	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		return nil, fmt.Errorf("creating tagged object: %s", diags.Errors()[0].Detail())
	}
	return obj, nil
}

var _ function.Function = (*RegEncodeFunction)(nil)

type RegEncodeFunction struct{}

func NewRegEncodeFunction() function.Function {
	return &RegEncodeFunction{}
}

func (f *RegEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regencode"
}

func (f *RegEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as a Windows .reg file",
		MarkdownDescription: "Encodes a Terraform object as a Windows Registry Editor export (.reg) file (Version 5). " +
			"The input must be a map of registry key paths to maps of value names. " +
			"Plain strings become REG_SZ values. Tagged objects from regdword(), regqword(), etc. " +
			"are converted to their native registry types. " +
			"Pass an optional options object with a \"comments\" key to add ; comments.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "An object of {\"HKEY_...\\\\Path\" = {\"ValueName\" = value}} to encode as a .reg file.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name: "options",
			Description: "An optional options object. Supported keys: " +
				"\"comments\" (object) — mirrored structure where string values become ; comments above the matching key or value. " +
				"Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

func (f *RegEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}

	var comments attr.Value

	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("options must be an object, got %T", optsArgs[0].UnderlyingValue())))
			return
		}
		if c, ok := obj.Attributes()["comments"]; ok {
			comments = c
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	obj, ok := value.UnderlyingValue().(basetypes.ObjectValue)
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must be an object with registry key paths as keys."))
		return
	}

	root := regis3.NewKeyEntry(nil, "")

	paths := make([]string, 0, len(obj.Attributes()))
	for p := range obj.Attributes() {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		valuesAttr := obj.Attributes()[path]
		valuesObj, ok := valuesAttr.(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Key %q must be an object of values.", path)))
			return
		}

		key := root.FindOrCreateKey(path)

		for vName, vAttr := range valuesObj.Attributes() {
			regName := vName
			if regName == "@" {
				regName = "" // Default value
			}

			entry := key.FindOrCreateValue(regName)
			if err := setRegValue(entry, vAttr); err != nil {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Key %q value %q: %s", path, vName, err.Error())))
				return
			}
		}
	}

	var buf bytes.Buffer
	writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptions{}, false)
	if err := writer.Write(&buf, root); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to write .reg file: "+err.Error()))
		return
	}

	result := buf.String()

	// Apply comments by injecting ; lines into the output.
	if comments != nil {
		result = applyRegComments(result, comments)
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}

// applyRegComments injects ; comment lines into the .reg file output.
// The comments value mirrors the data structure: top-level keys are registry
// paths, nested keys are value names, string values become ; comments.
func applyRegComments(output string, comments attr.Value) string {
	commentsObj, ok := comments.(basetypes.ObjectValue)
	if !ok {
		return output
	}

	lines := strings.Split(output, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a key path line: [HKEY_...]
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			keyPath := trimmed[1 : len(trimmed)-1]
			// Look for a comment on this key path.
			if commentVal, ok := commentsObj.Attributes()[keyPath]; ok {
				if sv, ok := commentVal.(basetypes.StringValue); ok {
					result = appendRegComment(result, sv.ValueString())
				}
			}
			result = append(result, line)
			continue
		}

		// Check if this is a value line: "Name"=... or @=...
		valueName := extractRegValueName(trimmed)
		if valueName != "" {
			// Find which key section we're in by looking backwards for [KEY_PATH].
			keyPath := findCurrentKeyPath(result)
			if keyPath != "" {
				if keyComments, ok := commentsObj.Attributes()[keyPath]; ok {
					if keyCommentsObj, ok := keyComments.(basetypes.ObjectValue); ok {
						if commentVal, ok := keyCommentsObj.Attributes()[valueName]; ok {
							if sv, ok := commentVal.(basetypes.StringValue); ok {
								result = appendRegComment(result, sv.ValueString())
							}
						}
					}
				}
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// appendRegComment adds ; comment lines to the output.
func appendRegComment(lines []string, comment string) []string {
	for _, cline := range strings.Split(comment, "\n") {
		lines = append(lines, "; "+cline)
	}
	return lines
}

// extractRegValueName extracts the value name from a .reg value line.
// Returns "" if the line is not a value line.
func extractRegValueName(line string) string {
	if strings.HasPrefix(line, "@=") {
		return "@"
	}
	if strings.HasPrefix(line, "\"") {
		// Find closing quote.
		end := strings.Index(line[1:], "\"")
		if end >= 0 {
			return line[1 : end+1]
		}
	}
	return ""
}

// findCurrentKeyPath looks backwards through lines for the most recent [KEY_PATH].
func findCurrentKeyPath(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return ""
}

// setRegValue sets a registry ValueEntry from a Terraform attr.Value.
func setRegValue(entry *regis3.ValueEntry, v attr.Value) error {
	switch val := v.(type) {
	case basetypes.StringValue:
		entry.SetString(val.ValueString())
		return nil

	case basetypes.ObjectValue:
		return setRegValueFromTaggedObject(entry, val.Attributes())

	default:
		return fmt.Errorf("unsupported value type %T (use a string or tagged object from regdword(), regbinary(), etc.)", v)
	}
}

// setRegValueFromTaggedObject reads __reg_type and value from a tagged object.
func setRegValueFromTaggedObject(entry *regis3.ValueEntry, attrs map[string]attr.Value) error {
	typeAttr, hasType := attrs[regTypeKey]
	valueAttr, hasValue := attrs[regValueKey]
	if !hasType || !hasValue {
		return fmt.Errorf("tagged object must have %q and %q keys", regTypeKey, regValueKey)
	}

	typeStr, ok := typeAttr.(basetypes.StringValue)
	if !ok {
		return fmt.Errorf("%q must be a string", regTypeKey)
	}

	switch typeStr.ValueString() {
	case regTypeDword:
		sv, ok := valueAttr.(basetypes.StringValue)
		if !ok {
			return fmt.Errorf("dword value must be a string")
		}
		n, err := strconv.ParseUint(sv.ValueString(), 10, 32)
		if err != nil {
			return fmt.Errorf("invalid dword value %q: %w", sv.ValueString(), err)
		}
		entry.SetDword(uint32(n))

	case regTypeQword:
		sv, ok := valueAttr.(basetypes.StringValue)
		if !ok {
			return fmt.Errorf("qword value must be a string")
		}
		n, err := strconv.ParseUint(sv.ValueString(), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid qword value %q: %w", sv.ValueString(), err)
		}
		entry.SetQword(n)

	case regTypeBinary:
		sv, ok := valueAttr.(basetypes.StringValue)
		if !ok {
			return fmt.Errorf("binary value must be a hex string")
		}
		data, err := hex.DecodeString(sv.ValueString())
		if err != nil {
			return fmt.Errorf("invalid hex in binary value: %w", err)
		}
		entry.SetBinaryType(regis3.RegBinary, data)

	case regTypeMultiSz:
		var strs []string
		switch lv := valueAttr.(type) {
		case basetypes.TupleValue:
			for _, elem := range lv.Elements() {
				sv, ok := elem.(basetypes.StringValue)
				if !ok {
					return fmt.Errorf("multi_sz elements must be strings")
				}
				strs = append(strs, sv.ValueString())
			}
		case basetypes.ListValue:
			for _, elem := range lv.Elements() {
				sv, ok := elem.(basetypes.StringValue)
				if !ok {
					return fmt.Errorf("multi_sz elements must be strings")
				}
				strs = append(strs, sv.ValueString())
			}
		default:
			return fmt.Errorf("multi_sz value must be a list of strings")
		}
		entry.SetMultiString(strs)

	case regTypeExpandSz:
		sv, ok := valueAttr.(basetypes.StringValue)
		if !ok {
			return fmt.Errorf("expand_sz value must be a string")
		}
		entry.SetExpandString(sv.ValueString())

	case regTypeNone:
		sv, ok := valueAttr.(basetypes.StringValue)
		if !ok {
			return fmt.Errorf("none value must be a hex string")
		}
		data, err := hex.DecodeString(sv.ValueString())
		if err != nil {
			return fmt.Errorf("invalid hex in none value: %w", err)
		}
		entry.SetBinaryType(regis3.RegNone, data)

	case regTypeDelete:
		entry.SetRemoveFlag(true)

	default:
		return fmt.Errorf("unknown registry type %q", typeStr.ValueString())
	}

	return nil
}
