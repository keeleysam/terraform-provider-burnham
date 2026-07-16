/*
Ed25519 helpers: deterministic key derivation from a seed.

`ed25519_key_from_seed` is the Terraform-facing function: hand it any seed, get back a PEM-encoded PKCS#8 Ed25519 private key. Same input always produces the same key.

Ed25519 is naturally deterministic by spec (RFC 8032 §5.1.6, where the signing scalar and nonce are derived from the private-key seed plus the message), so unlike the ECDSA path there's no RFC 6979 wrapper needed. The `ed25519.PrivateKey` returned by `ed25519.NewKeyFromSeed` already satisfies `crypto.Signer` with deterministic behaviour, and both `x509.CreateCertificate` and `digitorus/pkcs7.SignWithoutAttr` produce byte-stable output when handed it directly.
*/

package cryptography

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"golang.org/x/crypto/hkdf"
)

const ed25519KeyFromSeedInfo = "burnham/ed25519_key_from_seed"

var _ function.Function = (*Ed25519KeyFromSeedFunction)(nil)

type Ed25519KeyFromSeedFunction struct{}

func NewEd25519KeyFromSeedFunction() function.Function { return &Ed25519KeyFromSeedFunction{} }

func (f *Ed25519KeyFromSeedFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ed25519_key_from_seed"
}

func (f *Ed25519KeyFromSeedFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Derive a deterministic Ed25519 private key from a seed (PEM PKCS#8 output)",
		MarkdownDescription: fmt.Sprintf("Stretches `seed` to 32 bytes with HKDF-SHA256 (info string `%q`), uses the result as the Ed25519 private-key seed per [RFC 8032 §5.1.5](https://www.rfc-editor.org/rfc/rfc8032#section-5.1.5), and returns the resulting key as PEM PKCS#8.\n\nDeterministic by construction: same `seed` → same key, every time. Pair with [`x509_self_sign`](#function-x509_self_sign) and [`pkcs7_sign`](#function-pkcs7_sign): both accept either ECDSA P-256 or Ed25519 keys and dispatch the right signing algorithm on the key type.\n\n```\nprovider::burnham::ed25519_key_from_seed(sha512(file(\"input.bin\")))\n→ \"-----BEGIN PRIVATE KEY-----\\nMC4CAQAwBQYDK2VwBCIEI…\\n-----END PRIVATE KEY-----\\n\"\n```\n\nEd25519 is naturally deterministic by spec (the per-signature nonce is HMAC-derived from the private key and the message itself), so unlike the [`ecdsa_p256_key_from_seed`](#function-ecdsa_p256_key_from_seed) path there is no RFC 6979 wrapper involved; the stdlib `crypto/ed25519` signer is already byte-stable.\n\n**Compatibility note.** Ed25519 in CMS / X.509 is supported by OpenSSL and the rest of the modern PKI ecosystem ([RFC 8032](https://www.rfc-editor.org/rfc/rfc8032), [RFC 8410](https://www.rfc-editor.org/rfc/rfc8410), [RFC 8419](https://www.rfc-editor.org/rfc/rfc8419)) but is **not accepted** by Apple's macOS configuration-profile installer at the keychain-import layer as of macOS 26.5; signed `.mobileconfig` files using Ed25519 install-time-fail. For Apple configuration profiles use [`ecdsa_p256_key_from_seed`](#function-ecdsa_p256_key_from_seed) instead. Ed25519 is the better choice when the signature consumer is anything else (OpenSSL `cms`, GPG-replacement workflows, container signing, internal tooling).\n\n%s", ed25519KeyFromSeedInfo, hclByteHandlingGotcha),
		Parameters: []function.Parameter{
			function.StringParameter{Name: "seed", Description: fmt.Sprintf("Input keying material (raw bytes). Any length works; HKDF stretches to the 32 bytes Ed25519 needs. Must not be empty and must not exceed %d bytes (%d MiB). For cryptographic security pass at least 16 bytes of high-entropy input.", signingSeedMaxBytes, signingSeedMaxBytes/(1024*1024))},
		},
		Return: function.StringReturn{},
	}
}

func (f *Ed25519KeyFromSeedFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var seed string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &seed))
	if resp.Error != nil {
		return
	}
	if len(seed) == 0 {
		resp.Error = function.NewArgumentFuncError(0, "seed must not be empty")
		return
	}
	if len(seed) > signingSeedMaxBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("seed exceeds maximum length: %d bytes; got %d", signingSeedMaxBytes, len(seed)))
		return
	}

	priv, err := ed25519KeyFromSeed([]byte(seed))
	if err != nil {
		resp.Error = function.NewFuncError("key derivation failed: " + err.Error())
		return
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		resp.Error = function.NewFuncError("PKCS#8 marshal failed: " + err.Error())
		return
	}
	out := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ed25519KeyFromSeed implements the seed → 32-byte Ed25519 seed reduction. Exposed at package scope so x509_self_sign / pkcs7_sign internal tests can build keys without going through the public function surface.
func ed25519KeyFromSeed(seed []byte) (ed25519.PrivateKey, error) {
	raw := make([]byte, ed25519.SeedSize) // 32 bytes
	if _, err := io.ReadFull(hkdf.New(sha256.New, seed, nil, []byte(ed25519KeyFromSeedInfo)), raw); err != nil {
		return nil, fmt.Errorf("HKDF expand: %w", err)
	}
	return ed25519.NewKeyFromSeed(raw), nil
}
