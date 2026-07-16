package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type IPToMixedNotationFunction struct{}

func NewIPToMixedNotationFunction() function.Function { return &IPToMixedNotationFunction{} }

func (f *IPToMixedNotationFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_to_mixed_notation"
}

//go:embed descriptions/ip_to_mixed_notation.md
var ipToMixedNotationDescription string

func (f *IPToMixedNotationFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Format an IPv6 address using mixed (dual) x:x:x:x:x:x:d.d.d.d notation",
		MarkdownDescription: ipToMixedNotationDescription,
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

//go:embed descriptions/ipv4_to_ipv4_mapped.md
var ipv4ToIpv4MappedDescription string

func (f *IPv4ToIPv4MappedFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert an IPv4 address to its IPv4-mapped IPv6 form (::ffff:d.d.d.d)",
		MarkdownDescription: ipv4ToIpv4MappedDescription,
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
