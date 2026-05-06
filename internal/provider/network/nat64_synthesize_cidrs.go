package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64SynthesizeCIDRsFunction struct{}

func NewNAT64SynthesizeCIDRsFunction() function.Function { return &NAT64SynthesizeCIDRsFunction{} }

func (f *NAT64SynthesizeCIDRsFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_synthesize_cidrs"
}

func (f *NAT64SynthesizeCIDRsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Convert a list of IPv4 CIDRs to NAT64 IPv6 CIDRs",
		MarkdownDescription: "Returns each IPv4 CIDR in `ipv4_cidrs` converted to its IPv6 equivalent " +
			"under `nat64_prefix`. Only /64 and /96 NAT64 prefixes are supported.\n\n" +
			"By default returns addresses in mixed `x:x:x:x:x:x:d.d.d.d/N` notation. " +
			"Pass `true` as the optional third argument to use standard hex notation.\n\n" +
			"**Common uses:** translating an entire IPv4 address plan into NAT64 IPv6 space in one call; " +
			"generating the full set of IPv6 pool CIDRs for a NAT64 service; building dual-stack " +
			"firewall allowlists that include both the IPv4 and NAT64-IPv6 forms of the same ranges.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "ipv4_cidrs",
				Description: "The list of IPv4 CIDRs to convert.",
				ElementType: types.StringType,
			},
			function.StringParameter{Name: "nat64_prefix", Description: "The NAT64 prefix (/64 or /96)."},
		},
		VariadicParameter: function.BoolParameter{
			Name:        "use_hex",
			Description: "Pass true to return standard hex IPv6 notation instead of mixed notation (default: false).",
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *NAT64SynthesizeCIDRsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ipv4CIDRs []string
	var nat64Prefix string
	var useHexArgs []bool
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv4CIDRs, &nat64Prefix, &useHexArgs))
	if resp.Error != nil {
		return
	}
	useMixed := !optionalArg(useHexArgs, false)

	result, err := iputil.NAT64SynthesizeCIDRs(ipv4CIDRs, nat64Prefix, useMixed)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
