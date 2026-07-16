/*
base64zopfli: a drop-in replacement for Terraform's built-in base64gzip that swaps the standard DEFLATE encoder for Zopfli's iterative one.

Zopfli (https://github.com/google/zopfli) spends far more CPU than zlib searching for a smaller DEFLATE encoding of the same data, but the bitstream it emits is ordinary RFC 1951 DEFLATE: any gunzip / zcat / compress-gzip decoder reads it without knowing or caring that Zopfli produced it. That is the whole value proposition: consumers of base64zopfli output change nothing, they just receive a slightly smaller gzip member.

We drive the pure-Go port github.com/foobaz/go-zopfli to produce a *raw* DEFLATE stream and then wrap it in an RFC 1952 gzip container by hand. The port also ships a GzipCompress helper, but it hardcodes the OS header byte to 3 (Unix); the spec calls for OS=255 (unknown), which keeps the output from leaking the producing platform and is the RFC's portable sentinel. Assembling the container ourselves is the only way to control that byte, and it also lets us pin MTIME=0 for determinism (Terraform plan stability requires byte-identical output for identical input).
*/

package compression

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/foobaz/go-zopfli/zopfli"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// Iteration-count bounds for the optional `iterations` option. The default matches upstream Zopfli (and the spec); the ceiling is well past the point of diminishing returns and exists only to reject obviously-pathological inputs at plan time.
const (
	zopfliDefaultIterations = 15
	zopfliMinIterations     = 1
	zopfliMaxIterations     = 100000
)

// zopfliGzip compresses input with Zopfli and returns a complete, RFC 1952-conformant gzip member. iterations is Zopfli's optimization-pass count (higher = smaller, slower); the caller is responsible for range-validating it.
func zopfliGzip(input []byte, iterations int) ([]byte, error) {
	// DefaultOptions() already matches the spec: NumIterations=15, BlockSplitting=true, BlockSplittingLast=false, BlockType=DYNAMIC. We override only the iteration count. BlockSplitting must stay on: it accounts for most of Zopfli's edge over plain gzip.
	opts := zopfli.DefaultOptions()
	opts.NumIterations = iterations

	var deflate bytes.Buffer
	if err := zopfli.DeflateCompress(&opts, input, &deflate); err != nil {
		return nil, err
	}

	out := make([]byte, 0, 10+deflate.Len()+8)

	/*
		RFC 1952 §2.3 header, all fields fixed for deterministic, minimal, portable output:
		  ID1 ID2 = 0x1f 0x8b (gzip magic)
		  CM      = 8         (DEFLATE)
		  FLG     = 0         (no FNAME/FEXTRA/FCOMMENT/FHCRC/FTEXT)
		  MTIME   = 0         (must be zero: "current time" would churn every plan)
		  XFL     = 2         (compressor used maximum compression, accurate for Zopfli)
		  OS      = 255       (unknown: portable sentinel, doesn't leak the producing platform)
	*/
	out = append(out, 0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff)
	out = append(out, deflate.Bytes()...)

	// RFC 1952 §2.3.1 trailer: CRC32 of the uncompressed data then ISIZE (input length mod 2^32), both little-endian.
	out = binary.LittleEndian.AppendUint32(out, crc32.ChecksumIEEE(input))
	out = binary.LittleEndian.AppendUint32(out, uint32(len(input)))

	return out, nil
}

//go:embed descriptions/base64zopfli.md
var base64zopfliDescription string

var _ function.Function = (*Base64ZopfliFunction)(nil)

type Base64ZopfliFunction struct{}

func NewBase64ZopfliFunction() function.Function { return &Base64ZopfliFunction{} }

func (f *Base64ZopfliFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "base64zopfli"
}

func (f *Base64ZopfliFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "gzip-compress with Zopfli and base64-encode (a tighter, drop-in base64gzip)",
		MarkdownDescription: base64zopfliDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "The string to compress. Arbitrary bytes; matches base64gzip's input type. The empty string is allowed and produces a valid gzip member that decompresses to \"\".",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { iterations = number }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

// zopfliOptions parses the optional options object, returning the resolved iteration count or a function error if the object is malformed. Range validation happens in Run.
func zopfliOptions(opts []types.Dynamic) (int, *function.FuncError) {
	iterations := zopfliDefaultIterations
	attrs, ferr := optionsutil.SingleOptionsObject(opts, "{ iterations = 100 }")
	if ferr != nil {
		return 0, ferr
	}
	for k, val := range attrs {
		switch k {
		case "iterations":
			n, err := optionsutil.NumberAttrToInt(val)
			if err != nil {
				return 0, function.NewArgumentFuncError(1, "options.iterations must be a whole number: "+err.Error())
			}
			iterations = n
		default:
			return 0, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are iterations", k))
		}
	}
	return iterations, nil
}

func (f *Base64ZopfliFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &optsArgs))
	if resp.Error != nil {
		return
	}

	iterations, ferr := zopfliOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if iterations < zopfliMinIterations || iterations > zopfliMaxIterations {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("iterations must be in [%d, %d]; received %d", zopfliMinIterations, zopfliMaxIterations, iterations))
		return
	}

	gz, err := zopfliGzip([]byte(input), iterations)
	if err != nil {
		resp.Error = function.NewFuncError("zopfli compression failed: " + err.Error())
		return
	}

	out := base64.StdEncoding.EncodeToString(gz)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
