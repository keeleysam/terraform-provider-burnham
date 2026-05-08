package dataformat

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

var _ function.Function = (*HCLDecodeFunction)(nil)

type HCLDecodeFunction struct{}

func NewHCLDecodeFunction() function.Function { return &HCLDecodeFunction{} }

func (f *HCLDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hcldecode"
}

func (f *HCLDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse an arbitrary HCL document into a value",
		MarkdownDescription: "Parses an arbitrary [HCL2](https://github.com/hashicorp/hcl) document — a sequence of `key = value` attribute statements — and returns it as a Terraform object. Values are evaluated as static literals: numbers, strings, booleans, lists/tuples, and objects/maps work; references to variables, data sources, or function calls do **not** (there is no eval context).\n\nBlock syntax (`block_type \"label\" { ... }`) is **not** supported. Inputs containing top-level blocks are rejected with an error rather than silently dropped — use `hcldecode` only for attribute-only documents. For `.tfvars` files (which are themselves attribute-only HCL), `hcldecode` works fine; the built-in `provider::terraform::decode_tfvars` is an alternative tuned for that specific case.\n\n**Common uses:** parsing simple HCL configs vendored alongside Terraform modules, reading attribute-only config files, or round-tripping HCL fragments emitted by other tooling.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "An HCL document body.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *HCLDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(input), "input.hcl")
	if diags.HasErrors() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse HCL: "+diags.Error()))
		return
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Unexpected HCL body type"))
		return
	}

	if len(body.Blocks) > 0 {
		blockNames := make([]string, len(body.Blocks))
		for i, b := range body.Blocks {
			blockNames[i] = b.Type
		}
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf(
			"hcldecode does not support block syntax — found block(s): %v. Use it for attribute-only documents (key = value statements).",
			blockNames,
		)))
		return
	}

	out := make(map[string]interface{}, len(body.Attributes))
	keys := make([]string, 0, len(body.Attributes))
	for k := range body.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		attr := body.Attributes[k]
		ctyVal, evalDiags := attr.Expr.Value(nil)
		if evalDiags.HasErrors() {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Failed to evaluate attribute %q: %s", k, evalDiags.Error())))
			return
		}
		goVal, err := ctyToGo(ctyVal)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Attribute %q: %s", k, err.Error())))
			return
		}
		out[k] = goVal
	}

	tfVal, err := goMapToObject(out)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*HCLEncodeFunction)(nil)

type HCLEncodeFunction struct{}

func NewHCLEncodeFunction() function.Function { return &HCLEncodeFunction{} }

func (f *HCLEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hclencode"
}

func (f *HCLEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode an object as an HCL attribute body",
		MarkdownDescription: "Encodes a Terraform object as a sequence of HCL attribute statements (`key = value` lines), one per object member, in alphabetical key order. Nested objects render as HCL object literals (`{ ... }`); lists render as bracketed sequences; primitives render as their natural HCL representation.\n\nThis is **not** the same as `provider::terraform::encode_tfvars` — that built-in is `.tfvars`-specific. Use `hclencode` for emitting general-purpose HCL config files.\n\nOutput is formatted with `hclwrite.Format`, matching Terraform's canonical formatting.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "An object to render as HCL.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *HCLEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	prepared := goValueForJSONEncode(goVal)
	asMap, ok := prepared.(map[string]interface{})
	if !ok {
		// orderedMap is the prepared form for objects; fall back to the original map.
		switch m := goVal.(type) {
		case map[string]interface{}:
			asMap = m
		default:
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("hclencode requires an object, got %T", goVal))
			return
		}
	}

	file := hclwrite.NewEmptyFile()
	body := file.Body()
	keys := make([]string, 0, len(asMap))
	for k := range asMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !hclsyntax.ValidIdentifier(k) {
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("attribute name %q is not a valid HCL identifier", k))
			return
		}
		v := asMap[k]
		if v == nil {
			// hclwrite has no canonical rendering for a bare null, and ctyjson can't infer a type from JSON null. Render explicitly.
			body.SetAttributeRaw(k, hclwriteRawNull())
			continue
		}
		ctyVal, err := goToCty(v)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Attribute %q: %s", k, err.Error())))
			return
		}
		body.SetAttributeValue(k, ctyVal)
	}
	formatted := hclwrite.Format(file.Bytes())
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, string(formatted)))
}

// hclwriteRawNull produces an hclwrite.Tokens that renders as a bare `null` literal. We can't go through SetAttributeValue(cty.NullVal(cty.DynamicPseudoType)) because hclwrite refuses to render dynamic-typed nulls.
func hclwriteRawNull() hclwrite.Tokens {
	return hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte("null")}}
}

// ctyToGo round-trips a cty.Value through JSON to produce a Go value drawn from the JSON value space. ctyjson.SimpleJSONValue handles the type-impl details we'd otherwise duplicate.
func ctyToGo(v cty.Value) (interface{}, error) {
	if v.IsNull() {
		return nil, nil
	}
	bytes, err := ctyjson.Marshal(v, v.Type())
	if err != nil {
		return nil, fmt.Errorf("marshaling cty value: %w", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(bytes)))
	dec.UseNumber()
	var out interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding cty JSON: %w", err)
	}
	return out, nil
}

// goToCty converts a Go value drawn from the JSON value space to a cty.Value via JSON round-trip. ImpliedType walks the JSON to infer a concrete cty type, then Unmarshal produces the value.
func goToCty(v interface{}) (cty.Value, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return cty.NilVal, fmt.Errorf("marshaling: %w", err)
	}
	implied, err := ctyjson.ImpliedType(bytes)
	if err != nil {
		return cty.NilVal, fmt.Errorf("inferring cty type: %w", err)
	}
	val, err := ctyjson.Unmarshal(bytes, implied)
	if err != nil {
		return cty.NilVal, fmt.Errorf("decoding cty value: %w", err)
	}
	return val, nil
}
