package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NPTv6TranslateFunction struct{}

func NewNPTv6TranslateFunction() function.Function { return &NPTv6TranslateFunction{} }

func (f *NPTv6TranslateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nptv6_translate"
}

func (f *NPTv6TranslateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Translate an IPv6 address between NPTv6 prefix mappings (RFC 6296)",
		MarkdownDescription: "Translates `ipv6` from `from_prefix` to `to_prefix` using the checksum-neutral algorithm defined in RFC 6296. Both prefixes must be /48.\n\nThe first 48 bits are replaced with the new prefix. An adjustment is then applied to bytes 8–9 (the first word of the Interface Identifier) so that the one's complement sum of all 128 bits is preserved — this keeps transport-layer (TCP/UDP) checksums valid without packet rewriting.\n\n**Common uses:** computing the external address an internal host will appear as through an NPTv6 gateway (e.g. for DNS, ACL, or route configuration); reverse-translating an external address back to its internal form by swapping `from_prefix` and `to_prefix`.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "ipv6", Description: "The IPv6 address to translate."},
			function.StringParameter{Name: "from_prefix", Description: "The /48 prefix the address currently belongs to."},
			function.StringParameter{Name: "to_prefix", Description: "The /48 prefix to translate into."},
		},
		Return: function.StringReturn{},
	}
}

func (f *NPTv6TranslateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var ipv6, fromPrefix, toPrefix string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &ipv6, &fromPrefix, &toPrefix))
	if resp.Error != nil {
		return
	}

	result, err := iputil.NPTv6Translate(ipv6, fromPrefix, toPrefix)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
