/*
Base64 encode / decode (RFC 4648).

Core's `base64encode`/`base64decode` only speak standard, padded base64. `base64encode` here takes an options object selecting any of the four RFC 4648 variants — standard or URL-safe (§5) alphabet, padded or raw — and otherwise matches core when called with no options. `base64decode` is deliberately lenient: it accepts either alphabet and tolerates missing padding (and ignores ASCII whitespace), so it is a friction-free superset of core's stricter decoder and round-trips anything `base64encode` produces regardless of options.
*/

package encoding

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

func base64Encode(b []byte, urlSafe, padding bool) string {
	enc := base64.StdEncoding
	if urlSafe {
		enc = base64.URLEncoding
	}
	if !padding {
		enc = enc.WithPadding(base64.NoPadding)
	}
	return enc.EncodeToString(b)
}

// base64DecodeLenient accepts standard or URL-safe alphabets, padded or not,
// with ASCII whitespace anywhere. It normalizes to the raw-standard form and
// decodes that, so every variant a caller might paste in is handled.
func base64DecodeLenient(s string) ([]byte, error) {
	t := stripASCIIWhitespace(s)
	t = strings.ReplaceAll(t, "-", "+")
	t = strings.ReplaceAll(t, "_", "/")
	t = strings.TrimRight(t, "=")
	return base64.RawStdEncoding.DecodeString(t)
}

// base64EncodeOptions parses the optional { url_safe, padding } object.
func base64EncodeOptions(opts []types.Dynamic) (urlSafe, padding bool, ferr *function.FuncError) {
	urlSafe, padding = false, true
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ url_safe = true, padding = false }`)
	if ferr != nil {
		return false, true, ferr
	}
	for k, v := range attrs {
		switch k {
		case "url_safe", "padding":
			b, ok := v.(basetypes.BoolValue)
			if !ok || b.IsNull() {
				return false, true, function.NewArgumentFuncError(1, fmt.Sprintf("options.%s must be a bool", k))
			}
			if k == "url_safe" {
				urlSafe = b.ValueBool()
			} else {
				padding = b.ValueBool()
			}
		default:
			return false, true, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are url_safe, padding", k))
		}
	}
	return urlSafe, padding, nil
}

// ─── base64encode ───────────────────────────────────────────────

var _ function.Function = (*Base64EncodeFunction)(nil)

type Base64EncodeFunction struct{}

func NewBase64EncodeFunction() function.Function { return &Base64EncodeFunction{} }

func (f *Base64EncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "base64encode"
}

func (f *Base64EncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Base64-encode bytes (RFC 4648), with options for URL-safe and padding",
		MarkdownDescription: "Base64-encodes the input's bytes per [RFC 4648](https://www.rfc-editor.org/rfc/rfc4648). With no options it produces standard, padded base64 — identical to Terraform's built-in `base64encode`. The optional object selects the variant:\n\n- `url_safe` (bool, default `false`) — use the URL- and filename-safe alphabet (§5: `-` and `_` instead of `+` and `/`), as used by JWT/JOSE, OAuth PKCE, and webhooks.\n- `padding` (bool, default `true`) — emit `=` padding. Set `false` for the raw, unpadded form some APIs require.\n\nThe input is taken as raw bytes (the literal UTF-8 bytes of the string); to encode bytes held as hex, pass `hexdecode(var.x)`.\n\n```\nbase64encode(\"Hello\")                          → \"SGVsbG8=\"\nbase64encode(var.token, { url_safe = true, padding = false })\n```",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "The bytes to encode, taken as the raw UTF-8 bytes of the string."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Keys: `url_safe` (bool, default false) and `padding` (bool, default true).",
		},
		Return: function.StringReturn{},
	}
}

func (f *Base64EncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	urlSafe, padding, ferr := base64EncodeOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := base64Encode([]byte(input), urlSafe, padding)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ─── base64decode ───────────────────────────────────────────────

var _ function.Function = (*Base64DecodeFunction)(nil)

type Base64DecodeFunction struct{}

func NewBase64DecodeFunction() function.Function { return &Base64DecodeFunction{} }

func (f *Base64DecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "base64decode"
}

func (f *Base64DecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Base64-decode (RFC 4648), accepting any variant",
		MarkdownDescription: "Decodes base64 to its bytes, returned as a string of those raw bytes. Deliberately lenient: it accepts both the standard and the URL-safe (§5) alphabets, tolerates missing `=` padding, and ignores ASCII whitespace — so it is a friction-free superset of Terraform's built-in `base64decode` (which rejects URL-safe input) and round-trips anything `base64encode` produces regardless of its options.\n\nThe result is a byte string; for binary that isn't valid UTF-8 you will usually feed it into another function rather than printing it.\n\n```\nbase64decode(\"SGVsbG8\")   # unpadded, url-safe alphabet — both fine\n→ \"Hello\"\n```",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "Base64 in either alphabet, padded or not; ASCII whitespace ignored."},
		},
		Return: function.StringReturn{},
	}
}

func (f *Base64DecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	raw, err := base64DecodeLenient(input)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid base64 input: "+err.Error())
		return
	}
	out := string(raw)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
