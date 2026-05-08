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
		Summary:             "Format an IPv6 address using mixed (dual) x:x:x:x:x:x:d.d.d.d notation",
		MarkdownDescription: "Returns the IPv6 address formatted with the last 32 bits expressed as a dotted-decimal IPv4 address, e.g. `64:ff9b::192.0.2.1` instead of `64:ff9b::c000:201`. Zero-compression (::) is applied to the hex portion. IPv4 addresses are returned unchanged.\n\n**Common uses:** making NAT64 addresses human-readable in outputs and documentation; formatting IPv4-mapped addresses for display; expressing IPv4-compatible addresses in a form that makes the embedded IPv4 immediately visible.",
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}

type IPv4ToIPv4MappedFunction struct{}

func NewIPv4ToIPv4MappedFunction() function.Function { return &IPv4ToIPv4MappedFunction{} }

func (f *IPv4ToIPv4MappedFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ipv4_to_ipv4_mapped"
}

func (f *IPv4ToIPv4MappedFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert an IPv4 address to its IPv4-mapped IPv6 form (::ffff:d.d.d.d)",
		MarkdownDescription: "Returns the RFC 4291 IPv4-mapped IPv6 representation of an IPv4 address in mixed notation, e.g. `192.0.2.1` → `::ffff:192.0.2.1`.\n\n**Common uses:** configuring dual-stack listeners and sockets that accept both IPv4 and IPv6 connections; expressing IPv4 addresses in systems that require IPv6 format (e.g. some BGP implementations, certain firewall APIs); building allowlists that cover IPv4-mapped IPv6 representations of IPv4 addresses.",
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
