/*
ECDSA helpers — deterministic P-256 key derivation, and a `crypto.Signer` wrapper that turns any `*ecdsa.PrivateKey` into a RFC 6979 deterministic signer.

`ecdsa_p256_key_from_seed` is the Terraform-facing function: hand it any seed, get back a PEM-encoded PKCS#8 P-256 private key. The same input always produces the same key.

`detECDSASigner` is internal — used by `x509_self_sign` and `pkcs7_sign` to make stdlib's `x509.CreateCertificate` and `github.com/digitorus/pkcs7`'s `SignWithoutAttr` produce byte-identical output across runs. Both libraries call `signer.Sign(rand, digest, opts)`; we ignore `rand` and derive `k` via RFC 6979.
*/

package cryptography

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha1" //nolint:gosec // SHA-1 mapping retained only to interoperate with legacy crypto.Hash values; we never default to it.
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/nspcc-dev/rfc6979"
	"golang.org/x/crypto/hkdf"
)

const (
	ecdsaKeyFromSeedInfo = "burnham/ecdsa_p256_key_from_seed"

	// signingSeedMaxBytes caps the `seed` input to defeat adversarial multi-gigabyte payloads. Matches the 8 MiB cap on `asn1_decode.der_base64` — every legitimate seed (hash output, file digest, salt) is orders of magnitude smaller.
	signingSeedMaxBytes = 8 * 1024 * 1024
)

var _ function.Function = (*ECDSAP256KeyFromSeedFunction)(nil)

type ECDSAP256KeyFromSeedFunction struct{}

func NewECDSAP256KeyFromSeedFunction() function.Function { return &ECDSAP256KeyFromSeedFunction{} }

func (f *ECDSAP256KeyFromSeedFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ecdsa_p256_key_from_seed"
}

func (f *ECDSAP256KeyFromSeedFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Derive a deterministic ECDSA P-256 private key from a seed (PEM PKCS#8 output)",
		MarkdownDescription: fmt.Sprintf("Stretches `seed` to 48 bytes with HKDF-SHA256 (info string `%q`), reduces modulo (n-1) and adds 1 to land in [1, n-1] uniformly, and assembles the resulting scalar into a `secp256r1` private key. Output is PEM PKCS#8.\n\nDeterministic by construction: same `seed` → same key, every time. Useful when you want a stable signing identity that's derived from a checked-in secret or input artefact rather than randomly generated and stored.\n\n```\nprovider::burnham::ecdsa_p256_key_from_seed(sha512(file(\"input.bin\")))\n→ \"-----BEGIN PRIVATE KEY-----\\nMIGHAgEAM…\\n-----END PRIVATE KEY-----\\n\"\n```\n\nPair with [`x509_self_sign`](#function-x509_self_sign) and [`pkcs7_sign`](#function-pkcs7_sign) to build deterministic signing pipelines that are byte-stable across Terraform plans.\n\n%s", ecdsaKeyFromSeedInfo, hclByteHandlingGotcha),
		Parameters: []function.Parameter{
			function.StringParameter{Name: "seed", Description: fmt.Sprintf("Input keying material (raw bytes). Any length — HKDF stretches to the 48 bytes the scalar derivation needs. Must not be empty and must not exceed %d bytes (%d MiB). For cryptographic security pass at least 16 bytes of high-entropy input.", signingSeedMaxBytes, signingSeedMaxBytes/(1024*1024))},
		},
		Return: function.StringReturn{},
	}
}

func (f *ECDSAP256KeyFromSeedFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
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

	priv, err := ecdsaP256KeyFromSeed([]byte(seed))
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

// ecdsaP256KeyFromSeed implements the seed → P-256 scalar → key reduction. Exposed at package scope so x509_self_sign / pkcs7_sign internal tests can build keys without going through the public function surface.
func ecdsaP256KeyFromSeed(seed []byte) (*ecdsa.PrivateKey, error) {
	raw := make([]byte, 48)
	if _, err := io.ReadFull(hkdf.New(sha256.New, seed, nil, []byte(ecdsaKeyFromSeedInfo)), raw); err != nil {
		return nil, fmt.Errorf("HKDF expand: %w", err)
	}
	curve := elliptic.P256()
	n := curve.Params().N
	d := new(big.Int).Mod(new(big.Int).SetBytes(raw), new(big.Int).Sub(n, big.NewInt(1)))
	d.Add(d, big.NewInt(1))

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: curve},
		D:         d,
	}
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())
	return priv, nil
}

// detECDSASigner adapts an `*ecdsa.PrivateKey` to `crypto.Signer` with RFC 6979 deterministic `k`. Used internally by x509_self_sign and pkcs7_sign so the standard library's `x509.CreateCertificate` and the digitorus pkcs7 library — both of which take an opaque `crypto.Signer` — produce byte-identical output across runs.
//
// The `io.Reader` passed by callers is ignored; `k` is derived purely from the private scalar and the message digest per RFC 6979. The returned signature is DER-encoded `SEQUENCE { R, S }`, matching what `ecdsa.SignASN1` would emit.
//
// The hash used inside RFC 6979's HMAC is dispatched from `opts.HashFunc()` so it always matches the hash used to compute `digest`. RFC 6979 §3.2 requires both to be the same; using SHA-256 inside the HMAC against a SHA-384 digest would still produce a valid-shaped signature, but one that no other RFC 6979 implementation could reproduce — defeating the determinism guarantee.
type detECDSASigner struct{ priv *ecdsa.PrivateKey }

func (s *detECDSASigner) Public() crypto.PublicKey { return &s.priv.PublicKey }

func (s *detECDSASigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	h, err := hashForSignerOpts(opts, len(digest))
	if err != nil {
		return nil, err
	}
	r, sv := rfc6979.SignECDSA(s.priv, digest, h)
	return asn1.Marshal(struct{ R, S *big.Int }{r, sv})
}

// hashForSignerOpts maps a crypto.SignerOpts → a `hash.Hash` constructor for the HMAC inside RFC 6979. If opts is nil (some callers pass nil with the convention "the digest is whatever the digest length implies"), we infer from the digest length. Returns an error for unsupported / unknown hashes rather than silently picking SHA-256 — better to fail loudly than to emit signatures that disagree with every other RFC 6979 implementation.
func hashForSignerOpts(opts crypto.SignerOpts, digestLen int) (func() hash.Hash, error) {
	var alg crypto.Hash
	if opts != nil {
		alg = opts.HashFunc()
	}
	if alg == 0 {
		// Infer from digest length. Covers the case where callers (e.g. some PKCS#7 paths) pass nil opts.
		switch digestLen {
		case sha256.Size:
			alg = crypto.SHA256
		case sha512.Size384:
			alg = crypto.SHA384
		case sha512.Size:
			alg = crypto.SHA512
		case sha1.Size:
			alg = crypto.SHA1
		default:
			return nil, fmt.Errorf("cannot infer hash for digest length %d", digestLen)
		}
	}
	switch alg {
	case crypto.SHA256:
		return sha256.New, nil
	case crypto.SHA384:
		return sha512.New384, nil
	case crypto.SHA512:
		return sha512.New, nil
	case crypto.SHA1:
		return sha1.New, nil
	default:
		return nil, fmt.Errorf("unsupported hash %v", alg)
	}
}
