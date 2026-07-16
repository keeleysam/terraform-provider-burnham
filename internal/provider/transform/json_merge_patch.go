package transform

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//go:embed descriptions/json_merge_patch.md
var jsonMergePatchDescription string

var _ function.Function = (*JSONMergePatchFunction)(nil)

type JSONMergePatchFunction struct{}

func NewJSONMergePatchFunction() function.Function { return &JSONMergePatchFunction{} }

func (f *JSONMergePatchFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "json_merge_patch"
}

func (f *JSONMergePatchFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Apply an RFC 7396 JSON Merge Patch to a value",
		MarkdownDescription: jsonMergePatchDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The document to patch.",
			},
			function.DynamicParameter{
				Name:        "patch",
				Description: "A merge-patch document. Keys with non-null values overwrite the target; keys with null values delete the corresponding target key.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JSONMergePatchFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value, patch types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &patch))
	if resp.Error != nil {
		return
	}
	if unknownDynamicResultIfNeeded(ctx, resp, value.UnderlyingValue(), patch.UnderlyingValue()) {
		return
	}

	docGo, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	patchGo, err := terraformToJSON(patch.UnderlyingValue())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "failed to convert patch: "+err.Error())
		return
	}

	docBytes, err := json.Marshal(docGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to marshal document: "+err.Error()))
		return
	}

	patchBytes, err := json.Marshal(patchGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to marshal patch: "+err.Error()))
		return
	}

	mergedBytes, err := jsonpatch.MergePatch(docBytes, patchBytes)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "failed to apply merge patch: "+err.Error())
		return
	}

	var mergedGo interface{}
	dec := json.NewDecoder(bytes.NewReader(mergedBytes))
	dec.UseNumber()
	if err := dec.Decode(&mergedGo); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to decode merged document: "+err.Error()))
		return
	}

	tfVal, err := jsonToTerraform(mergedGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
