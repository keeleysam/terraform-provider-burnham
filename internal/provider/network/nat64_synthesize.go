package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
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
