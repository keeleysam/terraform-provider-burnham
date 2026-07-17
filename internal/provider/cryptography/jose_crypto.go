/*
Low-level JOSE crypto: compact-JWS signing and verification, plus PEM key parsing shared by the jwt_* functions.

The compact JWS format (RFC 7515 §3.1) is `base64url(header) . base64url(payload) . base64url(signature)`. We hand-roll sign and verify rather than delegating to a JOSE library's signer so we fully control determinism, in particular the ES256 fixed-length raw R||S encoding (RFC 7518 §3.4) produced from the RFC 6979 deterministic signer, and the exact bytes each algorithm signs.

Supported algorithms:

  - HS256 / HS384 / HS512: HMAC (deterministic by construction).
  - ES256: ECDSA P-256 with SHA-256, signed via the package's RFC 6979 detECDSASigner so identical inputs yield byte-identical signatures. The JWS signature is the fixed 64-byte R||S concatenation, not the ASN.1 DER SEQUENCE the signer emits, so we convert.
  - EdDSA: Ed25519 (deterministic by spec, RFC 8032).
  - RS256 / RS384 / RS512: RSASSA-PKCS1-v1_5 (deterministic).

RSASSA-PSS is intentionally omitted: its signatures are randomised, which would break the determinism guarantee this provider makes across the board.
*/

package cryptography

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"hash"
	"math/big"
)

// b64uEncode / b64uDecode are RFC 4648 §5 URL-safe base64 with no padding, the encoding JOSE uses for every segment (RFC 7515 §2).
func b64uEncode(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func b64uDecode(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

// jwsAlgHash maps a signing algorithm to the crypto.Hash it uses. The second return is false for algorithms whose signature is not over a pre-hash (none here) or that are unrecognised.
func jwsAlgHash(alg string) (crypto.Hash, bool) {
	switch alg {
	case "HS256", "ES256", "RS256":
		return crypto.SHA256, true
	case "HS384", "RS384":
		return crypto.SHA384, true
	case "HS512", "RS512":
		return crypto.SHA512, true
	case "EdDSA":
		// Ed25519 signs the message directly (PureEdDSA); there is no external pre-hash.
		return 0, true
	default:
		return 0, false
	}
}

func newHashByCrypto(h crypto.Hash) func() hash.Hash {
	switch h {
	case crypto.SHA256:
		return sha256.New
	case crypto.SHA384:
		return sha512.New384
	case crypto.SHA512:
		return sha512.New
	default:
		return nil
	}
}

// signJWS computes the JWS signature bytes over signingInput for the given algorithm. keyMaterial is the caller's raw key argument: the shared secret bytes for HS*, or a PEM-encoded private key for ES256 / EdDSA / RS*.
func signJWS(signingInput []byte, alg string, keyMaterial []byte) ([]byte, error) {
	switch alg {
	case "HS256", "HS384", "HS512":
		h, _ := jwsAlgHash(alg)
		if len(keyMaterial) == 0 {
			return nil, fmt.Errorf("%s key (HMAC secret) must not be empty", alg)
		}
		mac := hmac.New(newHashByCrypto(h), keyMaterial)
		mac.Write(signingInput)
		return mac.Sum(nil), nil

	case "ES256":
		priv, err := parseECPrivateKeyPEM(keyMaterial)
		if err != nil {
			return nil, err
		}
		if priv.Curve != elliptic.P256() {
			return nil, fmt.Errorf("ES256 requires a P-256 key, got %s", priv.Curve.Params().Name)
		}
		digest := sha256.Sum256(signingInput)
		// Reuse the RFC 6979 deterministic signer, then convert its ASN.1 DER SEQUENCE {R,S} to the fixed 64-byte R||S JWS encoding.
		der, err := (&detECDSASigner{priv: priv}).Sign(nil, digest[:], crypto.SHA256)
		if err != nil {
			return nil, fmt.Errorf("ES256 sign: %w", err)
		}
		r, s, err := ecdsaDERToRS(der)
		if err != nil {
			return nil, err
		}
		return ecdsaRSToRaw(r, s, ecdsaKeySize(priv.Curve)), nil

	case "EdDSA":
		priv, err := parseEd25519PrivateKeyPEM(keyMaterial)
		if err != nil {
			return nil, err
		}
		return ed25519.Sign(priv, signingInput), nil

	case "RS256", "RS384", "RS512":
		priv, err := parseRSAPrivateKeyPEM(keyMaterial)
		if err != nil {
			return nil, err
		}
		h, _ := jwsAlgHash(alg)
		digest := hashBytes(h, signingInput)
		sig, err := rsa.SignPKCS1v15(nil, priv, h, digest)
		if err != nil {
			return nil, fmt.Errorf("%s sign: %w", alg, err)
		}
		return sig, nil

	default:
		return nil, fmt.Errorf("unsupported algorithm %q (want one of HS256, HS384, HS512, ES256, EdDSA, RS256, RS384, RS512)", alg)
	}
}

// verifyJWS checks a JWS signature over signingInput. keyMaterial is the shared secret bytes for HS*, or a PEM public key / certificate / private key for the asymmetric algorithms.
func verifyJWS(signingInput, sig []byte, alg string, keyMaterial []byte) (bool, error) {
	switch alg {
	case "HS256", "HS384", "HS512":
		expected, err := signJWS(signingInput, alg, keyMaterial)
		if err != nil {
			return false, err
		}
		return hmac.Equal(expected, sig), nil

	case "ES256":
		pub, err := parseECPublicKeyPEM(keyMaterial)
		if err != nil {
			return false, err
		}
		size := ecdsaKeySize(pub.Curve)
		if len(sig) != 2*size {
			return false, nil
		}
		r := new(big.Int).SetBytes(sig[:size])
		s := new(big.Int).SetBytes(sig[size:])
		digest := sha256.Sum256(signingInput)
		return ecdsa.Verify(pub, digest[:], r, s), nil

	case "EdDSA":
		pub, err := parseEd25519PublicKeyPEM(keyMaterial)
		if err != nil {
			return false, err
		}
		return ed25519.Verify(pub, signingInput, sig), nil

	case "RS256", "RS384", "RS512":
		pub, err := parseRSAPublicKeyPEM(keyMaterial)
		if err != nil {
			return false, err
		}
		h, _ := jwsAlgHash(alg)
		digest := hashBytes(h, signingInput)
		return rsa.VerifyPKCS1v15(pub, h, digest, sig) == nil, nil

	default:
		return false, fmt.Errorf("unsupported algorithm %q", alg)
	}
}

func hashBytes(h crypto.Hash, b []byte) []byte {
	hh := h.New()
	hh.Write(b)
	return hh.Sum(nil)
}

func ecdsaKeySize(curve elliptic.Curve) int {
	return (curve.Params().BitSize + 7) / 8
}

// ecdsaRSToRaw serialises (r,s) as the fixed-length R||S JWS signature (RFC 7518 §3.4): each of r and s is left-padded with zeroes to the curve's coordinate byte length.
func ecdsaRSToRaw(r, s *big.Int, size int) []byte {
	out := make([]byte, 2*size)
	r.FillBytes(out[:size])
	s.FillBytes(out[size:])
	return out
}

// ecdsaDERToRS parses an ASN.1 DER `SEQUENCE { INTEGER r, INTEGER s }` (the shape ecdsa.SignASN1 / detECDSASigner emit) into its two integers.
func ecdsaDERToRS(der []byte) (r, s *big.Int, err error) {
	var parsed struct{ R, S *big.Int }
	rest, err := asn1.Unmarshal(der, &parsed)
	if err != nil {
		return nil, nil, fmt.Errorf("decode ECDSA DER signature: %w", err)
	}
	if len(rest) != 0 {
		return nil, nil, fmt.Errorf("trailing bytes after ECDSA DER signature")
	}
	return parsed.R, parsed.S, nil
}

// --- PEM key parsing --------------------------------------------------------

func parseECPrivateKeyPEM(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	der, err := firstPEMBlockBytes(string(pemBytes), "PRIVATE KEY", "EC PRIVATE KEY")
	if err != nil {
		return nil, fmt.Errorf("EC private key: %w", err)
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if ec, ok := k.(*ecdsa.PrivateKey); ok {
			return ec, nil
		}
		return nil, fmt.Errorf("EC private key: expected ECDSA, got %T", k)
	}
	ec, err := x509.ParseECPrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("EC private key: not a recognised ECDSA key (PKCS#8 or SEC1)")
	}
	return ec, nil
}

func parseEd25519PrivateKeyPEM(pemBytes []byte) (ed25519.PrivateKey, error) {
	der, err := firstPEMBlockBytes(string(pemBytes), "PRIVATE KEY")
	if err != nil {
		return nil, fmt.Errorf("Ed25519 private key: %w", err)
	}
	k, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("Ed25519 private key: %w", err)
	}
	ed, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("Ed25519 private key: expected Ed25519, got %T", k)
	}
	return ed, nil
}

func parseRSAPrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	der, err := firstPEMBlockBytes(string(pemBytes), "PRIVATE KEY", "RSA PRIVATE KEY")
	if err != nil {
		return nil, fmt.Errorf("RSA private key: %w", err)
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, fmt.Errorf("RSA private key: expected RSA, got %T", k)
	}
	rk, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("RSA private key: not a recognised RSA key (PKCS#8 or PKCS#1)")
	}
	return rk, nil
}

// publicKeyFromPEM extracts a public key from a PEM block: a PKIX `PUBLIC KEY`, a `CERTIFICATE`, or a private key (whose public part is derived). This lets jwt_verify accept whatever the caller has on hand.
func publicKeyFromPEM(pemBytes []byte) (crypto.PublicKey, error) {
	der, err := firstPEMBlockBytes(string(pemBytes), "PUBLIC KEY", "CERTIFICATE", "PRIVATE KEY", "EC PRIVATE KEY", "RSA PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		return pub, nil
	}
	if cert, err := x509.ParseCertificate(der); err == nil {
		return cert.PublicKey, nil
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if s, ok := k.(crypto.Signer); ok {
			return s.Public(), nil
		}
	}
	if ec, err := x509.ParseECPrivateKey(der); err == nil {
		return &ec.PublicKey, nil
	}
	if rk, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return &rk.PublicKey, nil
	}
	return nil, fmt.Errorf("no usable public key in PEM (want PUBLIC KEY, CERTIFICATE, or a private key)")
}

func parseECPublicKeyPEM(pemBytes []byte) (*ecdsa.PublicKey, error) {
	pub, err := publicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected an ECDSA public key, got %T", pub)
	}
	return ec, nil
}

func parseEd25519PublicKeyPEM(pemBytes []byte) (ed25519.PublicKey, error) {
	pub, err := publicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	ed, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected an Ed25519 public key, got %T", pub)
	}
	return ed, nil
}

func parseRSAPublicKeyPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	pub, err := publicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected an RSA public key, got %T", pub)
	}
	return rk, nil
}
