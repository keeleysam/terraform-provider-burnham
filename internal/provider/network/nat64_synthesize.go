package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64SynthesizeFunction struct{}

func NewNAT64SynthesizeFunction() function.Function { return &NAT64SynthesizeFunction{} }

func (f *NAT64SynthesizeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_synthesize"
}

//go:embed descriptions/nat64_synthesize.md
var nat64SynthesizeDescription string

func (f *NAT64SynthesizeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Synthesize a NAT64 IPv6 address from an IPv4 address (RFC 6052)",
		MarkdownDescription: nat64SynthesizeDescription,
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

//go:embed descriptions/nat64_synthesize_cidr.md
var nat64SynthesizeCidrDescription string

func (f *NAT64SynthesizeCIDRFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert a single IPv4 CIDR to its NAT64 IPv6 CIDR equivalent",
		MarkdownDescription: nat64SynthesizeCidrDescription,
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

//go:embed descriptions/nat64_synthesize_cidrs.md
var nat64SynthesizeCidrsDescription string

func (f *NAT64SynthesizeCIDRsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert a list of IPv4 CIDRs to NAT64 IPv6 CIDRs",
		MarkdownDescription: nat64SynthesizeCidrsDescription,
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
	var ipv4CIDRsArg types.List
	var nat64Prefix string
	var useHexArgs []bool
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv4CIDRsArg, &nat64Prefix, &useHexArgs))
	if resp.Error != nil {
		return
	}
	ipv4CIDRs, argErr := cidrListArg(ipv4CIDRsArg, 0, "ipv4_cidrs")
	if argErr != nil {
		resp.Error = argErr
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
