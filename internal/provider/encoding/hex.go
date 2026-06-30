/*
Hex encode / decode.

Terraform core ships no hex decoder at all, and the cryptography family's byte-oriented functions (`hmac`, `hkdf`) have had to tell callers to base64-wrap hex keys for want of one. `hexencode` / `hexdecode` close that gap: bytes ↔ lowercase hex. `hexdecode` is lenient — it accepts upper- or lower-case digits and ignores ASCII whitespace, so a wrapped or spaced dump round-trips.
*/

package encoding

import (
	"context"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

func hexEncode(b []byte) string { return hex.EncodeToString(b) }

func hexDecodeLenient(s string) ([]byte, error) {
	return hex.DecodeString(stripASCIIWhitespace(s))
}

// ─── hexencode ──────────────────────────────────────────────────

var _ function.Function = (*HexEncodeFunction)(nil)

type HexEncodeFunction struct{}

func NewHexEncodeFunction() function.Function { return &HexEncodeFunction{} }

func (f *HexEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hexencode"
}

func (f *HexEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode bytes as a lowercase hex string",
		MarkdownDescription: "Encodes the input's bytes as a lowercase hexadecimal string (two hex digits per byte).\n\nThe input is taken as raw bytes — the literal UTF-8 bytes of the string HCL hands the function. To hex-encode bytes you already hold as base64, pass `base64decode(var.x)`.\n\n```\nhexencode(\"Hi\")\n→ \"4869\"\n```",
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

var _ function.Function = (*HexDecodeFunction)(nil)

type HexDecodeFunction struct{}

func NewHexDecodeFunction() function.Function { return &HexDecodeFunction{} }

func (f *HexDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hexdecode"
}

func (f *HexDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a hex string to its bytes",
		MarkdownDescription: "Decodes a hexadecimal string to its bytes, returned as a string of those raw bytes. Lenient: upper- and lower-case digits are both accepted, and ASCII whitespace is ignored, so a spaced or line-wrapped dump decodes cleanly.\n\nThe result is a byte string; for binary that isn't valid UTF-8 you will usually feed it straight into another function (for example `hmac(\"sha256\", hexdecode(var.key_hex), var.msg)`) rather than printing it.\n\n```\nhexdecode(\"4869\")\n→ \"Hi\"\n```",
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
