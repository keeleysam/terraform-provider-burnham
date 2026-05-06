package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64SynthesizeFunction struct{}

func NewNAT64SynthesizeFunction() function.Function { return &NAT64SynthesizeFunction{} }

func (f *NAT64SynthesizeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_synthesize"
}

func (f *NAT64SynthesizeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Synthesize a NAT64 IPv6 address from an IPv4 address (RFC 6052)",
		MarkdownDescription: "Embeds `ipv4` into the given NAT64 `prefix` following the RFC 6052 byte layout " +
			"to produce the corresponding IPv6 address. The prefix must be /32, /40, /48, /56, /64, or /96.\n\n" +
			"By default returns the address in mixed `x:x:x:x:x:x:d.d.d.d` notation " +
			"(e.g. `64:ff9b::192.0.2.1`). Pass `true` as the optional third argument to get standard hex notation instead.\n\n" +
			"**Common uses:** pre-computing NAT64 pool members for DNS64 AAAA records, " +
			"configuring NAT64 gateway address pools, generating IPv6 addresses for IPv4-only services.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ipv4", Description: "The IPv4 address to embed."},
			function.StringParameter{Name: "prefix", Description: "The NAT64 prefix (/32, /40, /48, /56, /64, or /96)."},
		},
		VariadicParameter: function.BoolParameter{
			Name:        "use_hex",
			Description: "Pass true to return standard hex IPv6 notation instead of mixed notation (default: false).",
		},
		Return: function.StringReturn{},
	}
}

func (f *NAT64SynthesizeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ipv4, prefix string
	var useHexArgs []bool
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv4, &prefix, &useHexArgs))
	if resp.Error != nil {
		return
	}
	useMixed := !optionalArg(useHexArgs, false)

	result, err := iputil.NAT64Synthesize(ipv4, prefix, useMixed)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}

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
