package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRContainsFunction struct{}

func NewCIDRContainsFunction() function.Function { return &CIDRContainsFunction{} }

func (f *CIDRContainsFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidr_contains"
}

func (f *CIDRContainsFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Check whether a CIDR fully contains another IP or CIDR",
		MarkdownDescription: "Returns `true` if `cidr` fully contains `other`. " +
			"`other` may be a bare IP address or a CIDR string.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "cidr", Description: "The outer CIDR."},
			function.StringParameter{Name: "other", Description: "The IP or CIDR to test for containment."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *CIDRContainsFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidr, other string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidr, &other))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRContains(cidr, other)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
