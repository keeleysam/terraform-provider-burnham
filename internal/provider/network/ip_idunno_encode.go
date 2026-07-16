package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

var _ function.Function = (*IPIDunnoEncodeFunction)(nil)

type IPIDunnoEncodeFunction struct{}

func NewIPIDunnoEncodeFunction() function.Function { return &IPIDunnoEncodeFunction{} }

func (f *IPIDunnoEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_idunno_encode"
}

//go:embed descriptions/ip_idunno_encode.md
var ipIdunnoEncodeDescription string

func (f *IPIDunnoEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode an IP address as RFC 8771 I-DUNNO (Internationalized Deliberately Unreadable Network Notation)",
		MarkdownDescription: ipIdunnoEncodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "IPv4 (dotted-quad) or IPv6 (colon-hex) address. An IPv4-mapped IPv6 address (`::ffff:1.2.3.4`) is encoded in its full 128-bit IPv6 form; decoding normalises it back to IPv4."},
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
