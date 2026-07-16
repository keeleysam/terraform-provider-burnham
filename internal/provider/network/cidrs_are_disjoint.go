package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type CIDRsAreDisjointFunction struct{}

func NewCIDRsAreDisjointFunction() function.Function { return &CIDRsAreDisjointFunction{} }

func (f *CIDRsAreDisjointFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cidrs_are_disjoint"
}

//go:embed descriptions/cidrs_are_disjoint.md
var cidrsAreDisjointDescription string

func (f *CIDRsAreDisjointFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Check that no two CIDRs in a list overlap each other",
		MarkdownDescription: cidrsAreDisjointDescription,
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "cidrs",
				Description: "The list of CIDRs to check.",
				ElementType: types.StringType,
			},
		},
		Return: function.BoolReturn{},
	}
}

func (f *CIDRsAreDisjointFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var cidrsArg types.List
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &cidrsArg))
	if resp.Error != nil {
		return
	}
	cidrs, argErr := cidrListArg(cidrsArg, 0, "cidrs")
	if argErr != nil {
		resp.Error = argErr
		return
	}

	result, err := iputil.CIDRsAreDisjoint(cidrs)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
