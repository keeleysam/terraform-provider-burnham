package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NAT64PrefixValidFunction struct{}

func NewNAT64PrefixValidFunction() function.Function { return &NAT64PrefixValidFunction{} }

func (f *NAT64PrefixValidFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nat64_prefix_valid"
}

//go:embed descriptions/nat64_prefix_valid.md
var nat64PrefixValidDescription string

func (f *NAT64PrefixValidFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Validate that a prefix meets NAT64 requirements (RFC 6052)",
		MarkdownDescription: nat64PrefixValidDescription,
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
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
