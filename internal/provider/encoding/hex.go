/*
Hex encode / decode.

Terraform core ships no hex decoder at all, and the cryptography family's byte-oriented functions (`hmac`, `hkdf`) have had to tell callers to base64-wrap hex keys for want of one. `hexencode` / `hexdecode` close that gap: bytes ↔ lowercase hex. `hexdecode` is lenient, accepting upper- or lower-case digits and ignoring ASCII whitespace, so a wrapped or spaced dump round-trips.
*/

package encoding

import (
	"context"
	_ "embed"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

func hexEncode(b []byte) string { return hex.EncodeToString(b) }

func hexDecodeLenient(s string) ([]byte, error) {
	return hex.DecodeString(stripASCIIWhitespace(s))
}

// ─── hexencode ──────────────────────────────────────────────────

//go:embed descriptions/hexencode.md
var hexencodeDescription string

var _ function.Function = (*HexEncodeFunction)(nil)

type HexEncodeFunction struct{}

func NewHexEncodeFunction() function.Function { return &HexEncodeFunction{} }

func (f *HexEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hexencode"
}

func (f *HexEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode bytes as a lowercase hex string",
		MarkdownDescription: hexencodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "The bytes to encode, taken as the raw UTF-8 bytes of the string."},
		},
		Return: function.StringReturn{},
	}
}

func (f *HexEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	out := hexEncode([]byte(input))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ─── hexdecode ──────────────────────────────────────────────────

//go:embed descriptions/hexdecode.md
var hexdecodeDescription string

var _ function.Function = (*HexDecodeFunction)(nil)

type HexDecodeFunction struct{}

func NewHexDecodeFunction() function.Function { return &HexDecodeFunction{} }

func (f *HexDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hexdecode"
}

func (f *HexDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a hex string to its bytes",
		MarkdownDescription: hexdecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "A hex string (case-insensitive; ASCII whitespace ignored)."},
		},
		Return: function.StringReturn{},
	}
}

func (f *HexDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	raw, err := hexDecodeLenient(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid hex input: "+err.Error())
		return
	}
	out := string(raw)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
