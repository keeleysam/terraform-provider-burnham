package transform

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

var _ function.Function = (*JSONPatchFunction)(nil)

type JSONPatchFunction struct{}

func NewJSONPatchFunction() function.Function { return &JSONPatchFunction{} }



func (f *JSONPatchFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "json_patch"
}

func (f *JSONPatchFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Apply an RFC 6902 JSON Patch to a value",
		MarkdownDescription: "Applies an [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902) JSON Patch document to a Terraform value and returns the patched result. The patch is a tuple of operation objects — each with an `op` (`\"add\"`, `\"remove\"`, `\"replace\"`, `\"move\"`, `\"copy\"`, or `\"test\"`), a `path` (an [RFC 6901](https://www.rfc-editor.org/rfc/rfc6901) JSON Pointer), and operation-specific fields (`value`, `from`).\n\nOperations are applied in order. If any operation fails (including a failed `test`), the function returns an error and no partial state is produced.\n\nBacked by [evanphx/json-patch](https://github.com/evanphx/json-patch).",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The document to patch.",
			},
			function.DynamicParameter{
				Name:        "patch",
				Description: "A list of RFC 6902 operation objects.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *JSONPatchFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value, patch types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &patch))
	if resp.Error != nil {
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

	// json-patch operates on JSON byte streams; round-trip both inputs through encoding/json.
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

	decoded, err := jsonpatch.DecodePatch(patchBytes)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "invalid JSON Patch: "+err.Error())
		return
	}

	patchedBytes, err := decoded.Apply(docBytes)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, "failed to apply patch: "+err.Error())
		return
	}

	var patchedGo interface{}
	dec := json.NewDecoder(bytes.NewReader(patchedBytes))
	dec.UseNumber()
	if err := dec.Decode(&patchedGo); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to decode patched document: "+err.Error()))
		return
	}

	tfVal, err := jsonToTerraform(patchedGo)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("failed to convert result: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}
