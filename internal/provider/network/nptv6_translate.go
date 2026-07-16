package network

import (
	"context"
	_ "embed"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/keeleysam/terraform-burnham/internal/provider/network/iputil"
)

type NPTv6TranslateFunction struct{}

func NewNPTv6TranslateFunction() function.Function { return &NPTv6TranslateFunction{} }

func (f *NPTv6TranslateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nptv6_translate"
}

//go:embed descriptions/nptv6_translate.md
var nptv6TranslateDescription string

func (f *NPTv6TranslateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Translate an IPv6 address between NPTv6 prefix mappings (RFC 6296)",
		MarkdownDescription: nptv6TranslateDescription,
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
