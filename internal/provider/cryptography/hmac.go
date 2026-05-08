/*
HMAC keyed-hash message authentication code (RFC 2104).

Common at boundaries: signing webhook payloads, deriving stable per-tenant tokens, validating CSRF cookies. Up to now Terraform users had to either drop into `external` data sources, use a sidecar, or hand-roll something with `sha256()` + `replace()` that didn't actually compute HMAC.

`key` and `message` are interpreted as raw bytes (the framework gives us UTF-8 strings; for keys that are themselves hex- or base64-encoded, decode first). Output is hex-encoded — easy to compare in HCL and matches what `openssl dgst -hmac …` prints by default.
*/

package cryptography

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = (*HMACFunction)(nil)

type HMACFunction struct{}

func NewHMACFunction() function.Function { return &HMACFunction{} }

func (f *HMACFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hmac"
}

func (f *HMACFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Compute an HMAC (RFC 2104) over a message, returning hex",
		MarkdownDescription: fmt.Sprintf("Returns the [HMAC-`algorithm`](https://www.rfc-editor.org/rfc/rfc2104) of `message` keyed by `key`, hex-encoded.\n\n`algorithm` is one of:\n\n- `\"sha1\"` — RFC 2104 / FIPS 180-4 (legacy; do not pick for new designs)\n- `\"sha224\"`, `\"sha256\"`, `\"sha384\"`, `\"sha512\"` — FIPS 180-4 SHA-2 family\n- `\"sha512_224\"`, `\"sha512_256\"` — truncated SHA-512 variants\n\n```\nhmac(\"sha256\", \"super-secret\", \"payload\")\n→ \"3da88…\" (hex)\n```\n\n%s\n\nThis function is a derivation, not an MAC verifier — produce the expected MAC and `==`-compare in HCL.", hclByteHandlingGotcha),
		Parameters: []function.Parameter{
			function.StringParameter{Name: "algorithm", Description: "Hash algorithm: \"sha1\", \"sha224\", \"sha256\", \"sha384\", \"sha512\", \"sha512_224\", or \"sha512_256\"."},
			function.StringParameter{Name: "key", Description: "The HMAC key, as raw bytes."},
			function.StringParameter{Name: "message", Description: "The message to authenticate."},
		},
		Return: function.StringReturn{},
	}
}

// hashByName returns a constructor and the hash's output size for a stdlib hash matching the given name. Returns (nil, 0) for unknown names. We deliberately do not expose MD5: it is structurally broken for HMAC purposes and there's no reason to ship a footgun in 2026.
func hashByName(name string) (func() hash.Hash, int) {
	switch name {
	case "sha1":
		return sha1.New, sha1.Size
	case "sha224":
		return sha256.New224, sha256.Size224
	case "sha256":
		return sha256.New, sha256.Size
	case "sha384":
		return sha512.New384, sha512.Size384
	case "sha512":
		return sha512.New, sha512.Size
	case "sha512_224":
		return sha512.New512_224, sha512.Size224
	case "sha512_256":
		return sha512.New512_256, sha512.Size256
	}
	return nil, 0
}

func (f *HMACFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var algorithm, key, message string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &algorithm, &key, &message))
	if resp.Error != nil {
		return
	}
	h, _ := hashByName(algorithm)
	if h == nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("algorithm must be one of sha1, sha224, sha256, sha384, sha512, sha512_224, sha512_256; received %q", algorithm))
		return
	}
	mac := hmac.New(h, []byte(key))
	mac.Write([]byte(message))
	out := hex.EncodeToString(mac.Sum(nil))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
