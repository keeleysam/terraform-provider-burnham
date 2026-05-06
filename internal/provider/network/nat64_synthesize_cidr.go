package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64SynthesizeCIDRFunction struct{}

func NewNAT64SynthesizeCIDRFunction() function.Function { return &NAT64SynthesizeCIDRFunction{} }

func (f *NAT64SynthesizeCIDRFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_synthesize_cidr"
}

func (f *NAT64SynthesizeCIDRFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Convert a single IPv4 CIDR to its NAT64 IPv6 CIDR equivalent",
		MarkdownDescription: "Returns the IPv6 CIDR that corresponds to `ipv4_cidr` under the given NAT64 `prefix`. " +
			"Only /64 and /96 NAT64 prefixes are supported (those where IPv4 bits occupy a contiguous range " +
			"in the IPv6 address).\n\n" +
			"By default returns the result in mixed `x:x:x:x:x:x:d.d.d.d/N` notation. " +
			"Pass `true` as the optional third argument to use standard hex notation.\n\n" +
			"**Common uses:** pre-computing the IPv6 pool CIDR for a NAT64 gateway configuration; " +
			"expressing IPv4 address allocations in IPv6 space for dual-stack firewall rules.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ipv4_cidr", Description: "The IPv4 CIDR to convert."},
			function.StringParameter{Name: "nat64_prefix", Description: "The NAT64 prefix (/64 or /96)."},
		},
		VariadicParameter: function.BoolParameter{
			Name:        "use_hex",
			Description: "Pass true to return standard hex IPv6 notation instead of mixed notation (default: false).",
		},
		Return: function.StringReturn{},
	}
}

func (f *NAT64SynthesizeCIDRFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ipv4CIDR, nat64Prefix string
	var useHexArgs []bool
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv4CIDR, &nat64Prefix, &useHexArgs))
	if resp.Error != nil {
		return
	}
	useMixed := !optionalArg(useHexArgs, false)

	result, err := iputil.NAT64SynthesizeCIDR(ipv4CIDR, nat64Prefix, useMixed)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
