package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64ExtractFunction struct{}

func NewNAT64ExtractFunction() function.Function { return &NAT64ExtractFunction{} }

func (f *NAT64ExtractFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_extract"
}

func (f *NAT64ExtractFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Extract the embedded IPv4 address from a NAT64 IPv6 address (RFC 6052)",
		MarkdownDescription: "Recovers the IPv4 address from a NAT64 IPv6 address.\n\n" +
			"With no second argument, extracts the **last 32 bits** of the IPv6 address as a " +
			"dotted-decimal IPv4 string. This is correct for the overwhelming common case: the " +
			"Well-Known Prefix `64:ff9b::/96` and any other `/96` NAT64 prefix.\n\n" +
			"With an optional `nat64_prefix` argument (e.g. `\"2001:db8::/48\"`), uses the " +
			"RFC 6052 byte layout for that prefix length instead — needed for `/32`–`/64` " +
			"prefixes where the IPv4 bytes don't sit in the last 32 bits.\n\n" +
			"**Common uses:** reverse-mapping NAT64 addresses in flow logs or firewall hits " +
			"back to the original IPv4; ACL generation from IPv6 traffic records.",
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
