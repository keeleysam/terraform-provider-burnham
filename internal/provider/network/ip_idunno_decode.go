package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

var _ function.Function = (*IPIDunnoDecodeFunction)(nil)

type IPIDunnoDecodeFunction struct{}

func NewIPIDunnoDecodeFunction() function.Function { return &IPIDunnoDecodeFunction{} }

func (f *IPIDunnoDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_idunno_decode"
}

//go:embed descriptions/ip_idunno_decode.md
var ipIdunnoDecodeDescription string

func (f *IPIDunnoDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode an RFC 8771 I-DUNNO string back to its IPv4 or IPv6 address",
		MarkdownDescription: ipIdunnoDecodeDescription,
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
