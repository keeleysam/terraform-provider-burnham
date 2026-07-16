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
	_ "embed"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"golang.org/x/crypto/hkdf"
)

const ed25519KeyFromSeedInfo = "burnham/ed25519_key_from_seed"

var _ function.Function = (*Ed25519KeyFromSeedFunction)(nil)

//go:embed descriptions/ed25519_key_from_seed.md
var ed25519KeyFromSeedDescription string

type Ed25519KeyFromSeedFunction struct{}

func NewEd25519KeyFromSeedFunction() function.Function { return &Ed25519KeyFromSeedFunction{} }

func (f *Ed25519KeyFromSeedFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ed25519_key_from_seed"
}

func (f *Ed25519KeyFromSeedFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Derive a deterministic Ed25519 private key from a seed (PEM PKCS#8 output)",
		MarkdownDescription: fmt.Sprintf(ed25519KeyFromSeedDescription, ed25519KeyFromSeedInfo, hclByteHandlingGotcha),
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
