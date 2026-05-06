package dataformat

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*PlistDateFunction)(nil)

type PlistDateFunction struct{}

func NewPlistDateFunction() function.Function {
	return &PlistDateFunction{}
}

func (f *PlistDateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistdate"
}

func (f *PlistDateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Create a plist date value from an RFC 3339 timestamp",
		Description: "Returns a tagged object representing an NSDate plist value. When passed to plistencode, this produces a <date> element. The same tagged object format is returned by plistdecode for <date> elements, enabling seamless round-trips.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "rfc3339",
				Description: "An RFC 3339 timestamp string, e.g. \"2025-06-01T00:00:00Z\".",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PlistDateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	// Validate the timestamp is valid RFC 3339.
	_, err := time.Parse(time.RFC3339, input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(
			"Invalid RFC 3339 timestamp: "+err.Error()))
		return
	}

	obj, err := makePlistTaggedObject(plistTypeDate, input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}
