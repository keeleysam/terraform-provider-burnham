package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRsOverlapAnyFunction struct{}

func NewCIDRsOverlapAnyFunction() function.Function { return &CIDRsOverlapAnyFunction{} }

func (f *CIDRsOverlapAnyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidrs_overlap_any"
}

//go:embed descriptions/cidrs_overlap_any.md
var cidrsOverlapAnyDescription string

func (f *CIDRsOverlapAnyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Check whether any CIDR in one list overlaps with any CIDR in another",
		MarkdownDescription: cidrsOverlapAnyDescription,
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
	var aArg, bArg types.List
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &aArg, &bArg))
	if resp.Error != nil {
		return
	}
	a, argErr := cidrListArg(aArg, 0, "a")
	if argErr != nil {
		resp.Error = argErr
		return
	}
	b, argErr := cidrListArg(bArg, 1, "b")
	if argErr != nil {
		resp.Error = argErr
		return
	}

	result, err := iputil.CIDRsOverlapAny(a, b)
	if err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
