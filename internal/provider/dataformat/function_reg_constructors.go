package dataformat

import (
	"context"
	"encoding/hex"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ─── regdword ────────────────────────────────────────────────────

var _ function.Function = (*RegDwordFunction)(nil)

type RegDwordFunction struct{}

func NewRegDwordFunction() function.Function  { return &RegDwordFunction{} }
func (f *RegDwordFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regdword"
}
func (f *RegDwordFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Create a REG_DWORD registry value",
		MarkdownDescription: "Returns a tagged object representing a `REG_DWORD` (32-bit unsigned integer) registry value, for use inside a `regencode` payload.\n\n" +
			"Pass the value as a decimal integer between `0` and `4294967295`. HCL doesn't accept `0x...` literals; convert to decimal manually or use " +
			"`parseint(\"01020304\", 16)`.\n\n" +
			"**Common uses:** typed registry values in Group Policy / endpoint config — feature flags, integer thresholds, and status fields that " +
			"must be `REG_DWORD` rather than `REG_SZ`.",
		Parameters: []function.Parameter{function.NumberParameter{Name: "value", Description: "A 32-bit unsigned integer (0–4294967295)."}},
		Return:     function.DynamicReturn{},
	}
}
func (f *RegDwordFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	if value == nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must not be null."))
		return
	}
	n, _ := value.Uint64()
	obj, err := makeRegTaggedObject(regTypeDword, types.StringValue(strconv.FormatUint(n, 10)))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// ─── regqword ────────────────────────────────────────────────────

var _ function.Function = (*RegQwordFunction)(nil)

type RegQwordFunction struct{}

func NewRegQwordFunction() function.Function  { return &RegQwordFunction{} }
func (f *RegQwordFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regqword"
}
func (f *RegQwordFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Create a REG_QWORD registry value",
		MarkdownDescription: "Returns a tagged object representing a `REG_QWORD` (64-bit unsigned integer) registry value, for use inside a `regencode` payload.\n\n" +
			"**Common uses:** large numeric values in registry-driven config — file size limits, byte offsets, or any integer that exceeds `REG_DWORD`'s 32-bit range.",
		Parameters: []function.Parameter{function.NumberParameter{Name: "value", Description: "A 64-bit unsigned integer."}},
		Return:     function.DynamicReturn{},
	}
}
func (f *RegQwordFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	if value == nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Value must not be null."))
		return
	}
	n, _ := value.Uint64()
	obj, err := makeRegTaggedObject(regTypeQword, types.StringValue(strconv.FormatUint(n, 10)))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// ─── regbinary ───────────────────────────────────────────────────

var _ function.Function = (*RegBinaryFunction)(nil)

type RegBinaryFunction struct{}

func NewRegBinaryFunction() function.Function { return &RegBinaryFunction{} }
func (f *RegBinaryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regbinary"
}
func (f *RegBinaryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Create a REG_BINARY registry value",
		MarkdownDescription: "Returns a tagged object representing a `REG_BINARY` registry value, for use inside a `regencode` payload. " +
			"The input is a hex-encoded string (no separators, no `0x` prefix).\n\n" +
			"**Common uses:** binary blobs in Group Policy and app preferences — certificate hashes, packed structures, or pre-computed " +
			"configuration payloads consumed by Windows components.",
		Parameters: []function.Parameter{function.StringParameter{Name: "hex", Description: "Hex-encoded binary data (e.g. \"48656c6c6f\")."}},
		Return:     function.DynamicReturn{},
	}
}
func (f *RegBinaryFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	if _, err := hex.DecodeString(input); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Invalid hex string: "+err.Error()))
		return
	}
	obj, err := makeRegTaggedObject(regTypeBinary, types.StringValue(input))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// ─── regmulti ────────────────────────────────────────────────────

var _ function.Function = (*RegMultiFunction)(nil)

type RegMultiFunction struct{}

func NewRegMultiFunction() function.Function { return &RegMultiFunction{} }
func (f *RegMultiFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regmulti"
}
func (f *RegMultiFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Create a REG_MULTI_SZ registry value",
		MarkdownDescription: "Returns a tagged object representing a `REG_MULTI_SZ` (null-separated list of strings) registry value, for use inside a `regencode` payload.\n\n" +
			"**Common uses:** registry values that are inherently lists — search paths, allowlist/denylist entries, or any field where the consuming " +
			"Windows component expects multi-string semantics rather than a single delimited string.",
		Parameters: []function.Parameter{
			function.DynamicParameter{Name: "strings", Description: "A list of strings."},
		},
		Return: function.DynamicReturn{},
	}
}
func (f *RegMultiFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	// Extract strings from tuple or list.
	var elements []attr.Value
	switch v := input.UnderlyingValue().(type) {
	case types.Tuple:
		elements = v.Elements()
	case types.List:
		elements = v.Elements()
	default:
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Argument must be a list of strings."))
		return
	}

	strElems := make([]attr.Value, len(elements))
	strTypes := make([]attr.Type, len(elements))
	for i, elem := range elements {
		sv, ok := elem.(types.String)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("All elements must be strings."))
			return
		}
		strElems[i] = sv
		strTypes[i] = types.StringType
	}

	tuple := types.TupleValueMust(strTypes, strElems)

	obj, err := makeRegTaggedObject(regTypeMultiSz, tuple)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// ─── regexpandsz ─────────────────────────────────────────────────

var _ function.Function = (*RegExpandSzFunction)(nil)

type RegExpandSzFunction struct{}

func NewRegExpandSzFunction() function.Function { return &RegExpandSzFunction{} }
func (f *RegExpandSzFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regexpandsz"
}
func (f *RegExpandSzFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Create a REG_EXPAND_SZ registry value",
		MarkdownDescription: "Returns a tagged object representing a `REG_EXPAND_SZ` registry value, for use inside a `regencode` payload. " +
			"`REG_EXPAND_SZ` differs from `REG_SZ` in that the consuming Windows component expands `%VARIABLE%` references at lookup time.\n\n" +
			"**Common uses:** path values that must adapt per-user or per-machine (`%APPDATA%`, `%SystemRoot%`, `%USERPROFILE%`), or any " +
			"registry-driven config that needs to substitute environment variables when read.",
		Parameters: []function.Parameter{function.StringParameter{Name: "value", Description: "A string with %VARIABLE% references (e.g. \"%SystemRoot%\\\\system32\")."}},
		Return:     function.DynamicReturn{},
	}
}
func (f *RegExpandSzFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	obj, err := makeRegTaggedObject(regTypeExpandSz, types.StringValue(input))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}
