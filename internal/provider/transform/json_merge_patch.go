package transform

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

var _ function.Function = (*JSONMergePatchFunction)(nil)

type JSONMergePatchFunction struct{}

func NewJSONMergePatchFunction() function.Function { return &JSONMergePatchFunction{} }

func (f *JSONMergePatchFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "json_merge_patch"
}

func (f *JSONMergePatchFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Apply an RFC 7396 JSON Merge Patch to a value",
		MarkdownDescription: "Applies an [RFC 7396](https://www.rfc-editor.org/rfc/rfc7396) JSON Merge Patch to a Terraform value and returns the merged result. Unlike RFC 6902 (JSON Patch), the merge patch *is* a partial document with the same shape as the target — keys present in the patch override keys in the target, and a `null` value in the patch *deletes* the corresponding key from the target. Arrays are replaced wholesale; they aren't merged element-wise.\n\nThis is the right tool for environment overlays and Kubernetes-style strategic-merge-adjacent layering where most of your patch is just \"set these fields, remove that one.\" For element-level array edits or `test`-gated operations, use `json_patch` (RFC 6902) instead.\n\nBacked by [evanphx/json-patch](https://github.com/evanphx/json-patch).",
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

	docGo, err := terraformToJSON(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	patchGo, err := terraformToJSON(patch.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert patch: "+err.Error()))
		return
	}

	docBytes, err := json.Marshal(docGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to marshal document: "+err.Error()))
		return
	}

	patchBytes, err := json.Marshal(patchGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to marshal patch: "+err.Error()))
		return
	}

	mergedBytes, err := jsonpatch.MergePatch(docBytes, patchBytes)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to apply merge patch: "+err.Error()))
		return
	}

	var mergedGo interface{}
	dec := json.NewDecoder(bytes.NewReader(mergedBytes))
	dec.UseNumber()
	if err := dec.Decode(&mergedGo); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to decode merged document: "+err.Error()))
		return
	}

	tfVal, err := jsonToTerraform(mergedGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
