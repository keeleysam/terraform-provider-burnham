/*
Bitwise integer operations, the set Terraform's configuration language leaves out entirely: it has no bitwise AND / OR / XOR / NOT, no shifts, and no popcount.

Every function here is integer-only and rejects a non-integral or infinite argument with a clear error that names the offending value. All arithmetic runs through *big.Int, so arbitrary-precision integers work and nothing silently overflows int64 (a left shift by 100 or a popcount of 2^64 is exact).

Sign semantics: for `bit_and` / `bit_or` / `bit_xor` a negative operand is treated as an infinite two's-complement bit string, matching big.Int. That is well defined, but the common use case (combining flag bits) uses non-negative values, so negatives are allowed rather than encouraged. `bit_not` is width-parameterized on purpose: a width-less complement of an integer is infinite in two's-complement, so `bit_not(value, bits)` complements within an unsigned field of `bits` bits (result = value XOR (2^bits - 1)) and requires 0 <= value < 2^bits. `bit_shift_right` is an arithmetic shift: it floors toward negative infinity for a negative value, matching big.Int.Rsh. `popcount` requires a non-negative value, because a negative has infinitely many set bits in two's-complement.
*/

package numerics

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"math/bits"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// intArg validates that a single number argument is a finite integer and returns it as a *big.Int, attributing any error to argument index argIdx.
func intArg(f *big.Float, argIdx int64, name string) (*big.Int, *function.FuncError) {
	if f.IsInf() {
		return nil, function.NewArgumentFuncError(argIdx, fmt.Sprintf("%s must be a finite integer; received an infinite value", name))
	}
	if !f.IsInt() {
		return nil, function.NewArgumentFuncError(argIdx, fmt.Sprintf("%s must be an integer; received %s", name, f.Text('g', -1)))
	}
	i, _ := f.Int(nil)
	return i, nil
}

// intList reads a non-empty list(number) argument and returns each element as a *big.Int, rejecting empty lists and non-integral or infinite elements.
func intList(ctx context.Context, req function.RunRequest) ([]*big.Int, *function.FuncError) {
	var raw []*big.Float
	if err := req.Arguments.Get(ctx, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, function.NewArgumentFuncError(0, "numbers must contain at least one value; received an empty list")
	}
	out := make([]*big.Int, len(raw))
	for i, v := range raw {
		if v.IsInf() {
			return nil, function.NewArgumentFuncError(0, fmt.Sprintf("numbers[%d] is infinite; bitwise operations require integers", i))
		}
		if !v.IsInt() {
			return nil, function.NewArgumentFuncError(0, fmt.Sprintf("numbers[%d] must be an integer; received %s", i, v.Text('g', -1)))
		}
		iv, _ := v.Int(nil)
		out[i] = iv
	}
	return out, nil
}

// setInt writes a *big.Int result into the response as a Number.
func setInt(ctx context.Context, resp *function.RunResponse, out *big.Int) {
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, new(big.Float).SetInt(out)))
}

// foldListParam is the shared parameter shape for the folded list functions.
func foldListParam() []function.Parameter {
	return []function.Parameter{
		function.ListParameter{
			Name:        "numbers",
			Description: "A non-empty list of integers.",
			ElementType: types.NumberType,
		},
	}
}

// runFold reads the list, folds op over it starting from the first element, and stores the result.
func runFold(ctx context.Context, req function.RunRequest, resp *function.RunResponse, op func(z, x, y *big.Int) *big.Int) {
	xs, ferr := intList(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	acc := new(big.Int).Set(xs[0])
	for _, x := range xs[1:] {
		op(acc, acc, x)
	}
	setInt(ctx, resp, acc)
}

// ──────────────────────────────────────────────────────────────────────
// bit_and
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/bit_and.md
var bitAndDescription string

var _ function.Function = (*BitAndFunction)(nil)

type BitAndFunction struct{}

func NewBitAndFunction() function.Function { return &BitAndFunction{} }

func (f *BitAndFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_and"
}

func (f *BitAndFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Bitwise AND folded over a list of integers",
		MarkdownDescription: bitAndDescription,
		Parameters:          foldListParam(),
		Return:              function.NumberReturn{},
	}
}

func (f *BitAndFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	runFold(ctx, req, resp, (*big.Int).And)
}

// ──────────────────────────────────────────────────────────────────────
// bit_or
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/bit_or.md
var bitOrDescription string

var _ function.Function = (*BitOrFunction)(nil)

type BitOrFunction struct{}

func NewBitOrFunction() function.Function { return &BitOrFunction{} }

func (f *BitOrFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_or"
}

func (f *BitOrFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Bitwise OR folded over a list of integers",
		MarkdownDescription: bitOrDescription,
		Parameters:          foldListParam(),
		Return:              function.NumberReturn{},
	}
}

func (f *BitOrFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	runFold(ctx, req, resp, (*big.Int).Or)
}

// ──────────────────────────────────────────────────────────────────────
// bit_xor
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/bit_xor.md
var bitXorDescription string

var _ function.Function = (*BitXorFunction)(nil)

type BitXorFunction struct{}

func NewBitXorFunction() function.Function { return &BitXorFunction{} }

func (f *BitXorFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_xor"
}

func (f *BitXorFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Bitwise XOR folded over a list of integers",
		MarkdownDescription: bitXorDescription,
		Parameters:          foldListParam(),
		Return:              function.NumberReturn{},
	}
}

func (f *BitXorFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	runFold(ctx, req, resp, (*big.Int).Xor)
}

// ──────────────────────────────────────────────────────────────────────
// bit_not
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/bit_not.md
var bitNotDescription string

var _ function.Function = (*BitNotFunction)(nil)

type BitNotFunction struct{}

func NewBitNotFunction() function.Function { return &BitNotFunction{} }

func (f *BitNotFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_not"
}

func (f *BitNotFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Complement of a value within an unsigned field of a given bit width",
		MarkdownDescription: bitNotDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The value to complement; must satisfy 0 <= value < 2^bits."},
			function.NumberParameter{Name: "bits", Description: "The field width in bits; must be >= 1."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *BitNotFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var valueF, bitsF *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &valueF, &bitsF))
	if resp.Error != nil {
		return
	}
	value, ferr := intArg(valueF, 0, "value")
	if ferr != nil {
		resp.Error = ferr
		return
	}
	bitsInt, ferr := intArg(bitsF, 1, "bits")
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if bitsInt.Sign() <= 0 {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("bits must be >= 1; received %s", bitsInt.String()))
		return
	}
	if !bitsInt.IsInt64() {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("bits is too large: %s", bitsInt.String()))
		return
	}
	width := uint(bitsInt.Int64())
	// mask = 2^bits - 1, the all-ones field.
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), width), big.NewInt(1))
	if value.Sign() < 0 || value.Cmp(mask) > 0 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("value must satisfy 0 <= value < 2^%d (i.e. 0..%s); received %s", width, mask.String(), value.String()))
		return
	}
	setInt(ctx, resp, new(big.Int).Xor(value, mask))
}

// ──────────────────────────────────────────────────────────────────────
// bit_shift_left
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/bit_shift_left.md
var bitShiftLeftDescription string

var _ function.Function = (*BitShiftLeftFunction)(nil)

type BitShiftLeftFunction struct{}

func NewBitShiftLeftFunction() function.Function { return &BitShiftLeftFunction{} }

func (f *BitShiftLeftFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_shift_left"
}

func (f *BitShiftLeftFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Left shift: value << n",
		MarkdownDescription: bitShiftLeftDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The integer to shift."},
			function.NumberParameter{Name: "n", Description: "The number of bit positions to shift by; must be >= 0."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *BitShiftLeftFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	value, n, ferr := readShiftArgs(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	setInt(ctx, resp, new(big.Int).Lsh(value, n))
}

// ──────────────────────────────────────────────────────────────────────
// bit_shift_right
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/bit_shift_right.md
var bitShiftRightDescription string

var _ function.Function = (*BitShiftRightFunction)(nil)

type BitShiftRightFunction struct{}

func NewBitShiftRightFunction() function.Function { return &BitShiftRightFunction{} }

func (f *BitShiftRightFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_shift_right"
}

func (f *BitShiftRightFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Arithmetic right shift: value >> n (floors toward negative infinity)",
		MarkdownDescription: bitShiftRightDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The integer to shift."},
			function.NumberParameter{Name: "n", Description: "The number of bit positions to shift by; must be >= 0."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *BitShiftRightFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	value, n, ferr := readShiftArgs(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	setInt(ctx, resp, new(big.Int).Rsh(value, n))
}

// readShiftArgs validates the (value, n) pair shared by the two shift functions: both integers, n >= 0 and small enough to be a shift count.
func readShiftArgs(ctx context.Context, req function.RunRequest) (*big.Int, uint, *function.FuncError) {
	var valueF, nF *big.Float
	if err := req.Arguments.Get(ctx, &valueF, &nF); err != nil {
		return nil, 0, err
	}
	value, ferr := intArg(valueF, 0, "value")
	if ferr != nil {
		return nil, 0, ferr
	}
	n, ferr := intArg(nF, 1, "n")
	if ferr != nil {
		return nil, 0, ferr
	}
	if n.Sign() < 0 {
		return nil, 0, function.NewArgumentFuncError(1, fmt.Sprintf("n must be >= 0; received %s", n.String()))
	}
	if !n.IsInt64() {
		return nil, 0, function.NewArgumentFuncError(1, fmt.Sprintf("n is too large to use as a shift count: %s", n.String()))
	}
	return value, uint(n.Int64()), nil
}

// ──────────────────────────────────────────────────────────────────────
// popcount
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/popcount.md
var popcountDescription string

var _ function.Function = (*PopcountFunction)(nil)

type PopcountFunction struct{}

func NewPopcountFunction() function.Function { return &PopcountFunction{} }

func (f *PopcountFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "popcount"
}

func (f *PopcountFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Population count (Hamming weight): number of set bits",
		MarkdownDescription: popcountDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "A non-negative integer."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *PopcountFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var valueF *big.Float
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &valueF))
	if resp.Error != nil {
		return
	}
	value, ferr := intArg(valueF, 0, "value")
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if value.Sign() < 0 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("value must be >= 0 (a negative integer has infinitely many set bits in two's-complement); received %s", value.String()))
		return
	}
	count := 0
	for _, w := range value.Bits() {
		count += bits.OnesCount(uint(w))
	}
	setInt(ctx, resp, big.NewInt(int64(count)))
}

// ──────────────────────────────────────────────────────────────────────
// single-bit helpers: bit_set, bit_clear, bit_test
// ──────────────────────────────────────────────────────────────────────

// readValueIndex validates the (value, i) pair shared by the single-bit helpers: both integers, i >= 0 and usable as a bit index.
func readValueIndex(ctx context.Context, req function.RunRequest) (*big.Int, uint, *function.FuncError) {
	var valueF, iF *big.Float
	if err := req.Arguments.Get(ctx, &valueF, &iF); err != nil {
		return nil, 0, err
	}
	value, ferr := intArg(valueF, 0, "value")
	if ferr != nil {
		return nil, 0, ferr
	}
	i, ferr := intArg(iF, 1, "i")
	if ferr != nil {
		return nil, 0, ferr
	}
	if i.Sign() < 0 {
		return nil, 0, function.NewArgumentFuncError(1, fmt.Sprintf("i must be >= 0; received %s", i.String()))
	}
	if !i.IsInt64() {
		return nil, 0, function.NewArgumentFuncError(1, fmt.Sprintf("i is too large to use as a bit index: %s", i.String()))
	}
	return value, uint(i.Int64()), nil
}

//go:embed descriptions/bit_set.md
var bitSetDescription string

var _ function.Function = (*BitSetFunction)(nil)

type BitSetFunction struct{}

func NewBitSetFunction() function.Function { return &BitSetFunction{} }

func (f *BitSetFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_set"
}

func (f *BitSetFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Set bit i of value to 1",
		MarkdownDescription: bitSetDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The integer to modify."},
			function.NumberParameter{Name: "i", Description: "The zero-based bit index to set; must be >= 0."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *BitSetFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	value, i, ferr := readValueIndex(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	setInt(ctx, resp, new(big.Int).SetBit(value, int(i), 1))
}

//go:embed descriptions/bit_clear.md
var bitClearDescription string

var _ function.Function = (*BitClearFunction)(nil)

type BitClearFunction struct{}

func NewBitClearFunction() function.Function { return &BitClearFunction{} }

func (f *BitClearFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_clear"
}

func (f *BitClearFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Clear bit i of value to 0",
		MarkdownDescription: bitClearDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The integer to modify."},
			function.NumberParameter{Name: "i", Description: "The zero-based bit index to clear; must be >= 0."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *BitClearFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	value, i, ferr := readValueIndex(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	setInt(ctx, resp, new(big.Int).SetBit(value, int(i), 0))
}

//go:embed descriptions/bit_test.md
var bitTestDescription string

var _ function.Function = (*BitTestFunction)(nil)

type BitTestFunction struct{}

func NewBitTestFunction() function.Function { return &BitTestFunction{} }

func (f *BitTestFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "bit_test"
}

func (f *BitTestFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether bit i of value is set",
		MarkdownDescription: bitTestDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "value", Description: "The integer to test."},
			function.NumberParameter{Name: "i", Description: "The zero-based bit index to test; must be >= 0."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *BitTestFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	value, i, ferr := readValueIndex(ctx, req)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	set := value.Bit(int(i)) == 1
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, set))
}
