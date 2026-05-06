package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPv4ToIPv4MappedFunction struct{}

func NewIPv4ToIPv4MappedFunction() function.Function { return &IPv4ToIPv4MappedFunction{} }

func (f *IPv4ToIPv4MappedFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ipv4_to_ipv4_mapped"
}

func (f *IPv4ToIPv4MappedFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Convert an IPv4 address to its IPv4-mapped IPv6 form (::ffff:d.d.d.d)",
		MarkdownDescription: "Returns the RFC 4291 IPv4-mapped IPv6 representation of an IPv4 address " +
			"in mixed notation, e.g. `192.0.2.1` → `::ffff:192.0.2.1`.\n\n" +
			"**Common uses:** configuring dual-stack listeners and sockets that accept both IPv4 and IPv6 " +
			"connections; expressing IPv4 addresses in systems that require IPv6 format (e.g. some BGP " +
			"implementations, certain firewall APIs); building allowlists that cover IPv4-mapped IPv6 " +
			"representations of IPv4 addresses.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ipv4", Description: "The IPv4 address to convert."},
		},
		Return: function.StringReturn{},
	}
}

func (f *IPv4ToIPv4MappedFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ipv4 string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv4))
	if resp.Error != nil {
		return
	}

	result, err := iputil.IPv4ToIPv4Mapped(ipv4)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
