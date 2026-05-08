/*
PEM block decoding (RFC 7468).

Walks a PEM-armoured input — possibly containing multiple concatenated blocks — and returns each block's type label, its headers, and its base64-encoded body. The body is left base64-encoded (rather than hex or raw) because PEM is fundamentally a base64 envelope; round-tripping through `base64decode` keeps the bytes exact.
*/

package cryptography

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// pemMaxInputBytes / pemMaxBlocks bound the resources `pem_decode` will allocate. A multi-MB string of tiny `-----BEGIN X-----\n\n-----END X-----\n` blocks decodes to one Object + Map + handful of Strings per block; without a cap, a few hundred MB of input could allocate millions of attr.Value wrappers and OOM the provider. 16 MiB / 100k blocks is generous compared to anything realistic (a fullchain.pem is ≤ 16 KiB and ≤ 5 blocks).
const (
	pemMaxInputBytes = 16 * 1024 * 1024
	pemMaxBlocks     = 100_000
)

var _ function.Function = (*PEMDecodeFunction)(nil)

type PEMDecodeFunction struct{}

func NewPEMDecodeFunction() function.Function { return &PEMDecodeFunction{} }

func (f *PEMDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pem_decode"
}

// pemBlockAttrs is the per-block schema returned by pem_decode. type and base64_body are always present; headers maps every PEM header to its value (commonly empty for X.509 certs).
var pemBlockAttrs = map[string]attr.Type{
	"type":        types.StringType,
	"headers":     types.MapType{ElemType: types.StringType},
	"base64_body": types.StringType,
}

func (f *PEMDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode one or more PEM (RFC 7468) blocks into a list of {type, headers, base64_body} objects",
		MarkdownDescription: "Walks `pem` and returns a list, one entry per PEM block, of:\n\n- `type` — the block label between `-----BEGIN ` / `-----END ` (e.g. `\"CERTIFICATE\"`, `\"PRIVATE KEY\"`, `\"CERTIFICATE REQUEST\"`).\n- `headers` — `map(string)` of any RFC 1421 / 7468 header lines (often empty for modern PEM).\n- `base64_body` — the body, kept base64-encoded so the bytes round-trip exactly through `base64decode`. The body is the standard base64 alphabet, no line breaks.\n\nReturns an empty list when the input contains no PEM blocks. Garbage between blocks is silently skipped — same behaviour as `openssl` and most consumers.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pem", Description: "The PEM-armoured input. May contain multiple concatenated blocks."},
		},
		Return: function.ListReturn{ElementType: types.ObjectType{AttrTypes: pemBlockAttrs}},
	}
}

func (f *PEMDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	if len(input) > pemMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("pem input exceeds maximum supported length of %d bytes", pemMaxInputBytes))
		return
	}

	var blocks []attr.Value
	rest := []byte(input)
	for {
		block, next := pem.Decode(rest)
		if block == nil {
			break
		}
		if len(blocks) >= pemMaxBlocks {
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("pem input contains more than %d blocks", pemMaxBlocks))
			return
		}
		headers := map[string]attr.Value{}
		for k, v := range block.Headers {
			headers[k] = types.StringValue(v)
		}
		hdrs, diags := types.MapValue(types.StringType, headers)
		if diags.HasError() {
			resp.Error = function.NewFuncError("building headers map: " + diagsToString(diags))
			return
		}
		obj, diags := types.ObjectValue(pemBlockAttrs, map[string]attr.Value{
			"type":        types.StringValue(block.Type),
			"headers":     hdrs,
			"base64_body": types.StringValue(base64.StdEncoding.EncodeToString(block.Bytes)),
		})
		if diags.HasError() {
			resp.Error = function.NewFuncError("building block object: " + diagsToString(diags))
			return
		}
		blocks = append(blocks, obj)
		rest = next
	}

	out, diags := types.ListValue(types.ObjectType{AttrTypes: pemBlockAttrs}, blocks)
	if diags.HasError() {
		resp.Error = function.NewFuncError("building result list: " + diagsToString(diags))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
