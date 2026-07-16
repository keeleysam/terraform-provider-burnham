package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64ExtractFunction struct{}

func NewNAT64ExtractFunction() function.Function { return &NAT64ExtractFunction{} }

func (f *NAT64ExtractFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_extract"
}

//go:embed descriptions/nat64_extract.md
var nat64ExtractDescription string

func (f *NAT64ExtractFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Extract the embedded IPv4 address from a NAT64 IPv6 address (RFC 6052)",
		MarkdownDescription: nat64ExtractDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ipv6", Description: "The NAT64 IPv6 address to decode."},
		},
		VariadicParameter: function.StringParameter{
			Name:        "nat64_prefix",
			Description: "Optional: the NAT64 prefix used when the address was synthesized (e.g. \"2001:db8::/48\"). Required only for /32–/64 prefixes. Omit for the common /96 case.",
		},
		Return: function.StringReturn{},
	}
}

func (f *NAT64ExtractFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ipv6 string
	var prefixArgs []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv6, &prefixArgs))
	if resp.Error != nil {
		return
	}

	result, err := iputil.NAT64Extract(ipv6, optionalArg(prefixArgs, ""))
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
