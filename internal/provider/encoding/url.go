/*
URL percent-encoding / decoding.

Core's `urlencode` only does `application/x-www-form-urlencoded` (space → `+`), which is wrong for path segments, and core has **no** URL decoder at all. `urlencode` here takes a `mode` option: `query` (the form encoding, default, byte-identical to core), `path` (RFC 3986 path segment, space → `%20`), or `component` (strict: only RFC 3986 unreserved characters survive). `urldecode` fills the missing core function and takes the same `mode`, because `+` is ambiguous: it means a space in a query string but a literal `+` in a path. So `query` decode turns `+` into a space, while `path`/`component` decode leave `+` literal; both always decode `%XX`.
*/

package encoding

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// urlEncode percent-encodes s for the given mode (assumed already validated).
func urlEncode(s, mode string) string {
	switch mode {
	case "path":
		return url.PathEscape(s)
	case "component":
		return urlEncodeComponent(s)
	default: // "query"
		return url.QueryEscape(s)
	}
}

// urlEncodeComponent escapes every byte that is not an RFC 3986 unreserved
// character (`A-Za-z0-9-_.~`), with space → `%20`. This is slightly stricter
// than JS encodeURIComponent (which also leaves `!*'()`), making the result
// safe to drop into any URL position.
func urlEncodeComponent(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0f])
		}
	}
	return b.String()
}

// urlDecode reverses urlEncode for the given mode. query decode treats `+` as a
// space (form semantics); path/component decode leave `+` literal. Both decode
// `%XX` escapes.
func urlDecode(s, mode string) (string, error) {
	switch mode {
	case "path", "component":
		return url.PathUnescape(s)
	default: // "query"
		return url.QueryUnescape(s)
	}
}

// urlModeOption parses the optional { mode } object, defaulting to "query".
func urlModeOption(opts []types.Dynamic) (string, *function.FuncError) {
	mode := "query"
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ mode = "path" }`)
	if ferr != nil {
		return "", ferr
	}
	for k, v := range attrs {
		switch k {
		case "mode":
			s, ok := v.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return "", function.NewArgumentFuncError(1, "options.mode must be a string")
			}
			mode = s.ValueString()
			switch mode {
			case "query", "path", "component":
			default:
				return "", function.NewArgumentFuncError(1, fmt.Sprintf("options.mode must be one of \"query\", \"path\", \"component\"; received %q", mode))
			}
		default:
			return "", function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; the only supported key is mode", k))
		}
	}
	return mode, nil
}

const urlModeDescription = "An optional object. Key: `mode`, one of `\"query\"` (default; `application/x-www-form-urlencoded`, space ↔ `+`), `\"path\"` (RFC 3986 path segment, space ↔ `%20`, `+` literal), or `\"component\"` (strict: only unreserved characters unescaped)."

// ─── urlencode ──────────────────────────────────────────────────

var _ function.Function = (*URLEncodeFunction)(nil)

type URLEncodeFunction struct{}

func NewURLEncodeFunction() function.Function { return &URLEncodeFunction{} }

func (f *URLEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "urlencode"
}

func (f *URLEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Percent-encode a string for a URL, with a mode option",
		MarkdownDescription: "Percent-encodes a string for use in a URL. With no options it uses `mode = \"query\"` (`application/x-www-form-urlencoded`, encoding a space as `+`), which is byte-identical to Terraform's built-in `urlencode`. The optional `mode` selects where the value is going:\n\n- `\"query\"` (default): form encoding; space → `+`. For `a=b&c=d` query strings.\n- `\"path\"`: [RFC 3986](https://www.rfc-editor.org/rfc/rfc3986) path segment; space → `%20`, `/` escaped, `+` left literal. For building path components.\n- `\"component\"`: strict; everything except the unreserved set `A-Za-z0-9-_.~` is escaped, space → `%20`. For a value that must be safe in *any* URL position.\n\nCore's `urlencode` only does the `query` form, whose space → `+` is wrong inside a path; `path`/`component` fix that.\n\n```\nurlencode(\"a b/c\")                      → \"a+b%2Fc\"\nurlencode(\"a b/c\", { mode = \"path\" })   → \"a%20b%2Fc\"\n```",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "The string to percent-encode."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: urlModeDescription,
		},
		Return: function.StringReturn{},
	}
}

func (f *URLEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}
	mode, ferr := urlModeOption(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out := urlEncode(input, mode)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ─── urldecode ──────────────────────────────────────────────────

var _ function.Function = (*URLDecodeFunction)(nil)

type URLDecodeFunction struct{}

func NewURLDecodeFunction() function.Function { return &URLDecodeFunction{} }

func (f *URLDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "urldecode"
}

func (f *URLDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Percent-decode a URL-encoded string, with a mode option",
		MarkdownDescription: "Decodes a percent-encoded string, the function Terraform core is missing entirely. Both directions of `%XX` are decoded in every mode; the `mode` only controls how `+` is treated, because `+` is ambiguous (a space in a query string, a literal `+` in a path):\n\n- `\"query\"` (default): form semantics; `+` → space (and `%2B` → `+`). The inverse of `urlencode`'s default.\n- `\"path\"` / `\"component\"`: `+` is left literal; only `%XX` is decoded.\n\nThe result is a byte string; for input that decodes to non-UTF-8 bytes you will usually feed it into another function rather than printing it.\n\n```\nurldecode(\"a+b%2Fc\")                      → \"a b/c\"\nurldecode(\"1+1\", { mode = \"path\" })       → \"1+1\"\n```",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "input", Description: "The percent-encoded string to decode."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: urlModeDescription,
		},
		Return: function.StringReturn{},
	}
}

func (f *URLDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}
	mode, ferr := urlModeOption(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := urlDecode(input, mode)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid URL-encoded input: "+err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
