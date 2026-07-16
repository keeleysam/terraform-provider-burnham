/*
Base32 encode / decode (RFC 4648).

Terraform core has no base32 codec at all (a core PR adding one was closed as "belongs in a provider-defined function"). `base32encode` takes an options object for the two RFC 4648 alphabets, standard (`A–Z2–7`) and the extended-hex alphabet (`0–9A–V`, used by DNSSEC NSEC3), plus padding; with no options it produces standard, padded base32. `base32decode` is lenient: it uppercases, ignores ASCII whitespace, and tolerates missing padding. Unlike base64, the standard and hex alphabets overlap ambiguously, so the alphabet can't be auto-detected: `base32decode` takes the same `hex_alphabet` option (default standard). The classic use is TOTP/MFA secrets, which are unpadded standard base32.
*/

package encoding

import (
	"context"
	_ "embed"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

func base32Encoding(hexAlphabet, padding bool) *base32.Encoding {
	enc := base32.StdEncoding
	if hexAlphabet {
		enc = base32.HexEncoding
	}
	if !padding {
		enc = enc.WithPadding(base32.NoPadding)
	}
	return enc
}

func base32Encode(b []byte, hexAlphabet, padding bool) string {
	return base32Encoding(hexAlphabet, padding).EncodeToString(b)
}

// asciiToUpper uppercases only ASCII a–z, leaving every other byte untouched.
// Unicode case folding (strings.ToUpper) would fold letters like U+0131 (ı) and
// U+017F (ſ) into 'I' and 'S', which are in the base32 alphabet, so a rune-wise
// uppercase would let non-alphabet Unicode homoglyphs decode instead of erroring.
func asciiToUpper(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - ('a' - 'A')
		}
		return r
	}, s)
}

// base32DecodeLenient uppercases the input, drops ASCII whitespace, and tolerates
// missing padding. The alphabet (standard vs extended-hex) must be specified:
// the two overlap, so it cannot be auto-detected the way base64's disjoint
// alphabets can.
func base32DecodeLenient(s string, hexAlphabet bool) ([]byte, error) {
	t := asciiToUpper(stripASCIIWhitespace(s))
	t = strings.TrimRight(t, "=")
	enc := base32.StdEncoding
	if hexAlphabet {
		enc = base32.HexEncoding
	}
	return enc.WithPadding(base32.NoPadding).DecodeString(t)
}

// base32EncodeOptions parses the optional { hex_alphabet, padding } object.
func base32EncodeOptions(opts []types.Dynamic) (hexAlphabet, padding bool, ferr *function.FuncError) {
	hexAlphabet, padding = false, true
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ hex_alphabet = true, padding = false }`)
	if ferr != nil {
		return false, true, ferr
	}
	for k, v := range attrs {
		switch k {
		case "hex_alphabet", "padding":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() {
				return false, true, function.NewArgumentFuncError(1, fmt.Sprintf("options.%s must be a bool", k))
			}
			if k == "hex_alphabet" {
				hexAlphabet = b.ValueBool()
			} else {
				padding = b.ValueBool()
			}
		default:
			return false, true, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are hex_alphabet, padding", k))
		}
	}
	return hexAlphabet, padding, nil
}

// base32DecodeOptions parses the optional { hex_alphabet } object for decode.
func base32DecodeOptions(opts []types.Dynamic) (hexAlphabet bool, ferr *function.FuncError) {
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ hex_alphabet = true }`)
	if ferr != nil {
		return false, ferr
	}
	for k, v := range attrs {
		switch k {
		case "hex_alphabet":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() {
				return false, function.NewArgumentFuncError(1, "options.hex_alphabet must be a bool")
			}
			hexAlphabet = b.ValueBool()
		default:
			return false, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; the only supported key is hex_alphabet", k))
		}
	}
	return hexAlphabet, nil
}

// ─── base32encode ───────────────────────────────────────────────

//go:embed descriptions/base32encode.md
var base32encodeDescription string

var _ function.Function = (*Base32EncodeFunction)(nil)

type Base32EncodeFunction struct{}

func NewBase32EncodeFunction() function.Function { return &Base32EncodeFunction{} }

func (f *Base32EncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "base32encode"
}

func (f *Base32EncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Base32-encode bytes (RFC 4648), with options for alphabet and padding",
		MarkdownDescription: base32encodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "The bytes to encode, taken as the raw UTF-8 bytes of the string."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Keys: `hex_alphabet` (bool, default false) and `padding` (bool, default true).",
		},
		Return: function.StringReturn{},
	}
}

func (f *Base32EncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	hexAlphabet, padding, ferr := base32EncodeOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := base32Encode([]byte(input), hexAlphabet, padding)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ─── base32decode ───────────────────────────────────────────────

//go:embed descriptions/base32decode.md
var base32decodeDescription string

var _ function.Function = (*Base32DecodeFunction)(nil)

type Base32DecodeFunction struct{}

func NewBase32DecodeFunction() function.Function { return &Base32DecodeFunction{} }

func (f *Base32DecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "base32decode"
}

func (f *Base32DecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Base32-decode (RFC 4648), lenient on case and padding",
		MarkdownDescription: base32decodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "Base32 (case-insensitive, padding optional, ASCII whitespace ignored)."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Key: `hex_alphabet` (bool, default false), decode the extended-hex alphabet instead of standard.",
		},
		Return: function.StringReturn{},
	}
}

func (f *Base32DecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	hexAlphabet, ferr := base32DecodeOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	raw, err := base32DecodeLenient(input, hexAlphabet)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid base32 input: "+err.Error())
		return
	}
	out := string(raw)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
