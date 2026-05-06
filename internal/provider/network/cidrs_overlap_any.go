package network

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRsOverlapAnyFunction struct{}

func NewCIDRsOverlapAnyFunction() function.Function { return &CIDRsOverlapAnyFunction{} }

func (f *CIDRsOverlapAnyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidrs_overlap_any"
}

func (f *CIDRsOverlapAnyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Check whether any CIDR in one list overlaps with any CIDR in another",
		MarkdownDescription: "Returns `true` if any CIDR in list `a` overlaps with any CIDR in list `b`.\n\n" +
			"**Common uses:** pre-flight validation in `variable` validation blocks to ensure a proposed " +
			"VPC CIDR does not conflict with existing peered networks; checking that new security group " +
			"ranges don't collide with reserved address space.",
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "a",
				Description: "First list of CIDRs.",
				ElementType: types.StringType,
			},
			function.ListParameter{
				Name:        "b",
				Description: "Second list of CIDRs.",
				ElementType: types.StringType,
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *CIDRsOverlapAnyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b []string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}

	result, err := iputil.CIDRsOverlapAny(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
