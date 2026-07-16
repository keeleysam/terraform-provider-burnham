package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

var _ function.Function = (*IPIDunnoEncodeFunction)(nil)

type IPIDunnoEncodeFunction struct{}

func NewIPIDunnoEncodeFunction() function.Function { return &IPIDunnoEncodeFunction{} }

func (f *IPIDunnoEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_idunno_encode"
}

func (f *IPIDunnoEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode an IP address as RFC 8771 I-DUNNO (Internationalized Deliberately Unreadable Network Notation)",
		MarkdownDescription: "Encodes an IPv4 or IPv6 address into the Internationalized Deliberately Unreadable Network Notation per [RFC 8771](https://www.rfc-editor.org/rfc/rfc8771.html) (April 1, 2020). Output is a UTF-8 string of Unicode codepoints whose UTF-8 byte lengths carry the address bits per RFC §3 Table 1 (1-byte sequence = 7 bits, 2-byte = 11 bits, 3-byte = 16 bits, 4-byte = 21 bits).\n\nThe encoder is deterministic for a given input and reaches at least the **Minimum Confusion Level** of §4.1 (≥ 1 multi-octet UTF-8 sequence AND ≥ 1 IDNA2008-DISALLOWED character). RFC §5's worked example (`198.51.100.164` → `c\\u000Cl\\u04A4`, i.e. U+0063, U+000C, U+006C, U+04A4) round-trips through this encoder exactly.\n\nDual-stack: §3.1 specifies the bitstring length as \"32 bits for IPv4; 128 bits for IPv6\" and the rest of the spec operates on raw bits, so the same encoder handles both families.\n\n```\nprovider::burnham::ip_idunno_encode(\"198.51.100.164\")\n→ \"c\\u000Cl\\u04A4\"\n\nprovider::burnham::ip_idunno_encode(\"2001:db8::1\")\n→ some 8-codepoint UTF-8 string (output depends on the layout the encoder picks)\n```\n\nPair with [`ip_idunno_decode`](#function-ip_idunno_decode) to reverse the transformation. The RFC's §3.2 says deforming \"is intentionally omitted\" because \"humans SHOULD NOT attempt the process\"; the decoder is intended for the machines.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "IPv4 (dotted-quad) or IPv6 (colon-hex) address. Mapped IPv6 (`::ffff:1.2.3.4`) is normalised to its IPv4 form before encoding."},
		},
		Return: function.StringReturn{},
	}
}

func (f *IPIDunnoEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip))
	if resp.Error != nil {
		return
	}
	out, err := iputil.IDunnoEncode(ip)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
