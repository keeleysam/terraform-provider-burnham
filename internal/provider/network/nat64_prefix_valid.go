package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64PrefixValidFunction struct{}

func NewNAT64PrefixValidFunction() function.Function { return &NAT64PrefixValidFunction{} }

func (f *NAT64PrefixValidFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_prefix_valid"
}

func (f *NAT64PrefixValidFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Validate that a prefix meets NAT64 requirements (RFC 6052)",
		MarkdownDescription: "Returns `true` if `prefix` is a valid NAT64 prefix: it must be an IPv6 prefix " +
			"of length /32, /40, /48, /56, /64, or /96, and the reserved u-octet (bits 64–71) must be zero.\n\n" +
			"**Common uses:** `variable` validation blocks to reject operator-supplied NAT64 prefixes " +
			"that would produce malformed translated addresses.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "prefix", Description: "The prefix to validate."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *NAT64PrefixValidFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var prefix string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &prefix))
	if resp.Error != nil {
		return
	}

	result, err := iputil.NAT64PrefixValid(prefix)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
