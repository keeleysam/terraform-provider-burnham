package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPVersionFunction struct{}

func NewIPVersionFunction() function.Function { return &IPVersionFunction{} }

func (f *IPVersionFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_version"
}

func (f *IPVersionFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the IP version of an address",
		MarkdownDescription: "Returns `4` for IPv4 addresses and `6` for IPv6 addresses. IPv4-mapped IPv6 addresses (e.g. `::ffff:10.0.0.1`) are treated as IPv4.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "The IP address to inspect."},
		},
		Return: function.Int64Return{},
	}
}

func (f *IPVersionFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPVersion(ip)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
