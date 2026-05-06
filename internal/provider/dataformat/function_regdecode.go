package dataformat

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"

	"github.com/gersonkurz/go-regis3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		Description: "Decodes a Windows Registry Editor export (.reg) file into a Terraform object. " +
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
