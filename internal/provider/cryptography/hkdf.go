/*
HKDF — HMAC-based Key Derivation Function, RFC 5869.

Standard primitive for taking some input keying material (the `secret`) and stretching it into a deterministic byte stream of arbitrary length, optionally salted. Useful at plan time when you want to derive several tenant-specific values from a single master secret without storing them all.

The full Extract-then-Expand construction is supported via four parameters: hash algorithm, secret, salt, info, and output length. Returns hex-encoded bytes.
*/

package cryptography

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"golang.org/x/crypto/hkdf"
)

var _ function.Function = (*HKDFFunction)(nil)

type HKDFFunction struct{}

func NewHKDFFunction() function.Function { return &HKDFFunction{} }

func (f *HKDFFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hkdf"
}

func (f *HKDFFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "HKDF (RFC 5869) — derive `length` bytes from a secret, returning hex",
		MarkdownDescription: "Performs the [RFC 5869](https://www.rfc-editor.org/rfc/rfc5869) Extract-then-Expand HKDF construction:\n\n- `Extract`: PRK = HMAC-`algorithm`(`salt`, `secret`)\n- `Expand`: take `length` bytes from the output stream keyed by PRK and seeded with `info`\n\nAll byte-string inputs are interpreted as raw bytes (decode first if you have hex/base64). Output is hex-encoded.\n\n```\nhkdf(\"sha256\", \"input-keying-material\", \"salt\", \"per-tenant-foo\", 32)\n→ 64 hex chars (32 bytes)\n```\n\nUsed in TLS 1.3, the Signal protocol, and roughly every modern key-derivation pipeline. Backed by [`golang.org/x/crypto/hkdf`](https://pkg.go.dev/golang.org/x/crypto/hkdf).",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "algorithm", Description: "Hash algorithm — same set as `hmac`: \"sha1\", \"sha224\", \"sha256\", \"sha384\", \"sha512\", \"sha512_224\", \"sha512_256\"."},
			function.StringParameter{Name: "secret", Description: "Input keying material (raw bytes)."},
			function.StringParameter{Name: "salt", Description: "Optional salt (raw bytes). May be empty; RFC 5869 recommends a non-empty random salt where possible."},
			function.StringParameter{Name: "info", Description: "Optional context / application-specific info (raw bytes). May be empty."},
			function.Int64Parameter{Name: "length", Description: "Output length in bytes. Must be > 0 and at most 255 × hashLen."},
		},
		Return: function.StringReturn{},
	}
}

func (f *HKDFFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var algorithm, secret, salt, info string
	var length int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &algorithm, &secret, &salt, &info, &length))
	if resp.Error != nil {
		return
	}
	h, hashSize := hashByName(algorithm)
	if h == nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("algorithm must be one of sha1, sha224, sha256, sha384, sha512, sha512_224, sha512_256; received %q", algorithm))
		return
	}
	if length <= 0 {
		resp.Error = function.NewArgumentFuncError(4, fmt.Sprintf("length must be > 0; received %d", length))
		return
	}
	maxLen := int64(255 * hashSize)
	if length > maxLen {
		resp.Error = function.NewArgumentFuncError(4, fmt.Sprintf("length must be at most 255 × hashLen (%d for %s); received %d", maxLen, algorithm, length))
		return
	}

	r := hkdf.New(h, []byte(secret), []byte(salt), []byte(info))
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		resp.Error = function.NewFuncError("HKDF expand failed: " + err.Error())
		return
	}
	out := hex.EncodeToString(buf)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
