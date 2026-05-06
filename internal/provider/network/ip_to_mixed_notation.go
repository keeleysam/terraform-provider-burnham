package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPToMixedNotationFunction struct{}

func NewIPToMixedNotationFunction() function.Function { return &IPToMixedNotationFunction{} }

func (f *IPToMixedNotationFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_to_mixed_notation"
}

func (f *IPToMixedNotationFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Format an IPv6 address using mixed (dual) x:x:x:x:x:x:d.d.d.d notation",
		MarkdownDescription: "Returns the IPv6 address formatted with the last 32 bits expressed as a " +
			"dotted-decimal IPv4 address, e.g. `64:ff9b::192.0.2.1` instead of `64:ff9b::c000:201`. " +
			"Zero-compression (::) is applied to the hex portion. IPv4 addresses are returned unchanged.\n\n" +
			"**Common uses:** making NAT64 addresses human-readable in outputs and documentation; " +
			"formatting IPv4-mapped addresses for display; expressing IPv4-compatible addresses " +
			"in a form that makes the embedded IPv4 immediately visible.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ip", Description: "An IPv6 (or IPv4) address string."},
		},
		Return: function.StringReturn{},
	}
}

func (f *IPToMixedNotationFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ip string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ip))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPToMixedNotation(ip)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
