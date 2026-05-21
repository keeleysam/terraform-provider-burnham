package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

var _ function.Function = (*IPIDunnoDecodeFunction)(nil)

type IPIDunnoDecodeFunction struct{}

func NewIPIDunnoDecodeFunction() function.Function { return &IPIDunnoDecodeFunction{} }

func (f *IPIDunnoDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_idunno_decode"
}

func (f *IPIDunnoDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode an RFC 8771 I-DUNNO string back to its IPv4 or IPv6 address",
		MarkdownDescription: "Reverses [`ip_idunno_encode`](#function-ip_idunno_encode). Walks the input codepoint-by-codepoint, determines each codepoint's UTF-8 byte length, and accumulates the corresponding number of low-order bits per RFC 8771 §3 Table 1 (1-byte = 7 bits, 2-byte = 11, 3-byte = 16, 4-byte = 21). The total bit-payload disambiguates IPv4 (32–52 codepoint bits — 32 address + ≤ 20 padding) from IPv6 (128–148); those ranges don't overlap, so the decoder doesn't need a hint.\n\nReturns the address in canonical text form: dotted-quad for IPv4, [RFC 5952](https://www.rfc-editor.org/rfc/rfc5952.html) lowercase colon-hex for IPv6. Use `provider::burnham::ip_version(...)` if you need to branch on the family afterwards.\n\nRFC §3.2 says deforming \"is intentionally omitted. The machines will know how to do it, and by definition humans SHOULD NOT attempt the process.\" This is the machines knowing how to do it.\n\nErrors when the input isn't valid UTF-8, contains a surrogate codepoint, or has a total bit-payload that doesn't match either IPv4 or IPv6.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "encoded", Description: "An I-DUNNO string produced by `ip_idunno_encode` (or any RFC 8771-compliant encoder)."},
		},
		Return: function.StringReturn{},
	}
}

func (f *IPIDunnoDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var encoded string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &encoded))
	if resp.Error != nil {
		return
	}
	out, err := iputil.IDunnoDecode(encoded)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
