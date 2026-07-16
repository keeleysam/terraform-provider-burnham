package dataformat

import (
	_ "embed"

	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	regDwordMax uint64 = 0xFFFFFFFF         // 2^32 - 1
	regQwordMax uint64 = 0xFFFFFFFFFFFFFFFF // 2^64 - 1
)

// regUintFromNumber narrows a Terraform `*big.Float` argument to a non-negative integer in `[0, max]`. Returns a precise error message rather than relying on `(*big.Float).Uint64`'s silent saturation, which would otherwise turn `regdword(-1)` into `0` and `regdword(2^33)` into `MaxUint32`.
func regUintFromNumber(v *big.Float, paramName string, max uint64) (uint64, error) {
	if v == nil {
		return 0, fmt.Errorf("%s must not be null", paramName)
	}
	if v.IsInf() {
		return 0, fmt.Errorf("%s must be finite; received %s", paramName, v.Text('g', -1))
	}
	if !v.IsInt() {
		return 0, fmt.Errorf("%s must be a whole number; received %s", paramName, v.Text('g', -1))
	}
	if v.Sign() < 0 {
		return 0, fmt.Errorf("%s must be >= 0; received %s", paramName, v.Text('f', -1))
	}
	n, accuracy := v.Uint64()
	if accuracy != big.Exact || v.Cmp(new(big.Float).SetUint64(n)) != 0 {
		return 0, fmt.Errorf("%s must be in [0, %d]; received %s", paramName, max, v.Text('f', -1))
	}
	if n > max {
		return 0, fmt.Errorf("%s must be in [0, %d]; received %d", paramName, max, n)
	}
	return n, nil
}

// ─── regdword ────────────────────────────────────────────────────

var _ function.Function = (*RegDwordFunction)(nil)

//go:embed descriptions/regdword.md
var regdwordDescription string

type RegDwordFunction struct{}

func NewRegDwordFunction() function.Function { return &RegDwordFunction{} }
func (f *RegDwordFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regdword"
}
func (f *RegDwordFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Create a REG_DWORD registry value",
		MarkdownDescription: regdwordDescription,
		Parameters:          []function.Parameter{function.NumberParameter{Name: "value", Description: "A 32-bit unsigned integer (0–4294967295)."}},
		Return:              function.DynamicReturn{},
	}
}
func (f *RegDwordFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	n, err := regUintFromNumber(value, "value", regDwordMax)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	obj, err := makeRegTaggedObject(regTypeDword, types.StringValue(strconv.FormatUint(n, 10)))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// ─── regqword ────────────────────────────────────────────────────

var _ function.Function = (*RegQwordFunction)(nil)

//go:embed descriptions/regqword.md
var regqwordDescription string

type RegQwordFunction struct{}

func NewRegQwordFunction() function.Function { return &RegQwordFunction{} }
func (f *RegQwordFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regqword"
}
func (f *RegQwordFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Create a REG_QWORD registry value",
		MarkdownDescription: regqwordDescription,
		Parameters:          []function.Parameter{function.NumberParameter{Name: "value", Description: "A 64-bit unsigned integer (0–18446744073709551615)."}},
		Return:              function.DynamicReturn{},
	}
}
func (f *RegQwordFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}
	n, err := regUintFromNumber(value, "value", regQwordMax)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	obj, err := makeRegTaggedObject(regTypeQword, types.StringValue(strconv.FormatUint(n, 10)))
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

// ─── regbinary ───────────────────────────────────────────────────

var _ function.Function = (*RegBinaryFunction)(nil)

//go:embed descriptions/regbinary.md
var regbinaryDescription string

type RegBinaryFunction struct{}

func NewRegBinaryFunction() function.Function { return &RegBinaryFunction{} }
func (f *RegBinaryFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regbinary"
}
func (f *RegBinaryFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Create a REG_BINARY registry value",
		MarkdownDescription: regbinaryDescription,
		Parameters:          []function.Parameter{function.StringParameter{Name: "hex", Description: "Hex-encoded binary data (e.g. \"48656c6c6f\")."}},
		Return:              function.DynamicReturn{},
	}
}
func (f *RegBinaryFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	if input == "" {
		// An empty REG_BINARY value is technically valid in Windows but is almost always a bug at the Terraform layer (the user usually intended a missing or unset value, not a zero-byte payload). Reject explicitly so the mistake is loud.
		resp.Error = function.NewArgumentFuncError(0, "hex must not be empty; use a non-empty hex string or omit the value entirely")
		return
	}
	if _, err := hex.DecodeString(input); err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid hex string: "+err.Error())
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

//go:embed descriptions/regmulti.md
var regmultiDescription string

type RegMultiFunction struct{}

func NewRegMultiFunction() function.Function { return &RegMultiFunction{} }
func (f *RegMultiFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regmulti"
}
func (f *RegMultiFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Create a REG_MULTI_SZ registry value",
		MarkdownDescription: regmultiDescription,
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
		resp.Error = function.NewArgumentFuncError(0, "argument must be a list of strings")
		return
	}
	if len(elements) == 0 {
		// An empty REG_MULTI_SZ is legal in Windows but signals "I want no entries", which the encoder cannot distinguish from "I forgot to populate this list". Reject so the caller is forced to be explicit.
		resp.Error = function.NewArgumentFuncError(0, "strings must contain at least one entry; use an explicit empty registry value if you really want a zero-entry REG_MULTI_SZ")
		return
	}

	strElems := make([]attr.Value, len(elements))
	strTypes := make([]attr.Type, len(elements))
	for i, elem := range elements {
		sv, ok := elem.(types.String)
		if !ok {
			resp.Error = function.NewArgumentFuncError(0, "all elements must be strings")
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

//go:embed descriptions/regexpandsz.md
var regexpandszDescription string

type RegExpandSzFunction struct{}

func NewRegExpandSzFunction() function.Function { return &RegExpandSzFunction{} }
func (f *RegExpandSzFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "regexpandsz"
}
func (f *RegExpandSzFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Create a REG_EXPAND_SZ registry value",
		MarkdownDescription: regexpandszDescription,
		Parameters:          []function.Parameter{function.StringParameter{Name: "value", Description: "A string with %VARIABLE% references (e.g. \"%SystemRoot%\\\\system32\")."}},
		Return:              function.DynamicReturn{},
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
