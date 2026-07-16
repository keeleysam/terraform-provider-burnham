/*
base64brotli: Brotli compression (RFC 7932) followed by base64 encoding.

On text-heavy payloads Brotli at quality 11 produces noticeably smaller output than gzip (~8–10% on representative user_data), at the cost of requiring a brotli decompressor on the consuming side (`brotli -d`, shipped by every current Linux distro). It is the higher-effort sibling of base64zopfli: more savings, one-time consumer-side cost.

We use the pure-Go encoder github.com/molecule-man/go-brrr to keep the provider CGO_ENABLED=0 and trivially cross-compilable. It is byte-compatible with the RFC 7932 reference implementation and benchmarks at roughly 2.1× the one-shot throughput of github.com/andybalholm/brotli, which matters because user_data is compressed at quality 11 by default. The encoder is deterministic for a given input and options, which Terraform plan stability requires; there is no MTIME-equivalent in the Brotli format, so determinism needs no extra work beyond not feeding it any time- or randomness-derived input. We pass SizeHint = len(input): free since the whole input is already in memory, derived solely from the input so it preserves determinism, and it lets the encoder tune hasher and context-modeling decisions on large payloads.

Note on the RFC 7932 §10 encoder "mode" hint (text/generic/font): go-brrr's WriterOptions exposes no mode field at all, so there is no such knob to surface. Quality and window size (lgwin) are the knobs that actually change the output here.
*/

package compression

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
	brrr "github.com/molecule-man/go-brrr"
)

// Option bounds. Defaults favor maximum ratio because user_data is compressed once at plan time and decompressed many times. Ranges are exactly those of RFC 7932 / the brotli encoder.
const (
	brotliDefaultQuality = 11
	brotliMinQuality     = 0
	brotliMaxQuality     = 11

	brotliDefaultLgwin = 22
	brotliMinLgwin     = 10
	brotliMaxLgwin     = 24
)

// brotliCompress compresses input with Brotli and returns the raw RFC 7932 stream. quality is the 0..11 effort level and lgwin is the log2 window size (10..24); the caller is responsible for range-validating both.
func brotliCompress(input []byte, quality, lgwin int) ([]byte, error) {
	var buf bytes.Buffer
	w, err := brrr.NewWriterOptions(&buf, quality, brrr.WriterOptions{
		LGWin:    lgwin,
		SizeHint: uint(len(input)),
	})
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(input); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var _ function.Function = (*Base64BrotliFunction)(nil)

type Base64BrotliFunction struct{}

func NewBase64BrotliFunction() function.Function { return &Base64BrotliFunction{} }

func (f *Base64BrotliFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "base64brotli"
}

func (f *Base64BrotliFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Brotli-compress (RFC 7932) and base64-encode",
		MarkdownDescription: "Compresses `input` with [Brotli](https://www.rfc-editor.org/rfc/rfc7932) and returns the result as a base64-encoded brotli stream. On text-heavy payloads this is ~8–10% smaller than `base64gzip` (and a few percent smaller than `base64zopfli`), at the cost of requiring a brotli decompressor on the consuming side (`brotli -d`, shipped by every current Linux distro). Decompress with `base64 -d | brotli -d`, or any RFC 7932 decoder (browsers' `Content-Encoding: br`, Python `brotli`, etc.).\n\nThe encoder is deterministic for a given input and options (there is no MTIME-equivalent in the brotli format), so same `input` and options always produce byte-identical output, keeping plans stable.\n\nThe optional `options` object accepts:\n\n- `quality` (number): compression effort; default `11` (maximum ratio), range `[0, 11]`. Lower is faster with a worse ratio. Default is `11` because `user_data` is compressed once at plan time and decompressed many times.\n- `lgwin` (number): log₂ of the sliding-window size in bytes (RFC 7932 §9.1); default `22` (a 4 MiB window), range `[10, 24]`. Increase only for genuinely huge inputs with long-range repetition; decrease only if compress-time memory is constrained.\n\nThe RFC 7932 §10 encoder `mode` hint (text/generic/font) is intentionally **not** exposed: the pure-Go encoder this provider uses does not apply it (`text` and `generic` are byte-identical, `font` is unreachable through its public API), so a `mode` option would be a no-op rather than an honest knob.\n\n```\nboot_scripts_blob = provider::burnham::base64brotli(jsonencode(scripts))\nboot_scripts_blob = provider::burnham::base64brotli(jsonencode(scripts), { quality = 6 })\n```",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "The string to compress. Arbitrary bytes. The empty string is allowed and produces a valid brotli stream that decompresses to \"\".",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { quality = number, lgwin = number }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

// brotliOptions parses the optional options object, returning the resolved (quality, lgwin) or a function error if the object is malformed. Range validation happens in Run.
func brotliOptions(opts []types.Dynamic) (int, int, *function.FuncError) {
	quality := brotliDefaultQuality
	lgwin := brotliDefaultLgwin
	attrs, ferr := optionsutil.SingleOptionsObject(opts, "{ quality = 6 }")
	if ferr != nil {
		return 0, 0, ferr
	}
	for k, val := range attrs {
		switch k {
		case "quality":
			n, err := optionsutil.NumberAttrToInt(val)
			if err != nil {
				return 0, 0, function.NewArgumentFuncError(1, "options.quality must be a whole number: "+err.Error())
			}
			quality = n
		case "lgwin":
			n, err := optionsutil.NumberAttrToInt(val)
			if err != nil {
				return 0, 0, function.NewArgumentFuncError(1, "options.lgwin must be a whole number: "+err.Error())
			}
			lgwin = n
		default:
			return 0, 0, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are quality, lgwin", k))
		}
	}
	return quality, lgwin, nil
}

func (f *Base64BrotliFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}

	quality, lgwin, ferr := brotliOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if quality < brotliMinQuality || quality > brotliMaxQuality {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("quality must be in [%d, %d]; received %d", brotliMinQuality, brotliMaxQuality, quality))
		return
	}
	if lgwin < brotliMinLgwin || lgwin > brotliMaxLgwin {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("lgwin must be in [%d, %d]; received %d", brotliMinLgwin, brotliMaxLgwin, lgwin))
		return
	}

	compressed, err := brotliCompress([]byte(input), quality, lgwin)
	if err != nil {
		resp.Error = function.NewFuncError("brotli compression failed: " + err.Error())
		return
	}

	out := base64.StdEncoding.EncodeToString(compressed)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
