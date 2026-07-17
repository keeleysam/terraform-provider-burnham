package cryptography

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

// TestSignJWS_HS256_RFC7515A1 locks the HMAC + base64url signing path against the exact known-answer vector in RFC 7515 Appendix A.1. The signing input and expected signature are copied verbatim from the RFC.
func TestSignJWS_HS256_RFC7515A1(t *testing.T) {
	const (
		header     = "eyJ0eXAiOiJKV1QiLA0KICJhbGciOiJIUzI1NiJ9"
		payload    = "eyJpc3MiOiJqb2UiLA0KICJleHAiOjEzMDA4MTkzODAsDQogImh0dHA6Ly9leGFtcGxlLmNvbS9pc19yb290Ijp0cnVlfQ"
		keyB64u    = "AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"
		wantSigB64 = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	)
	key, err := base64.RawURLEncoding.DecodeString(keyB64u)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	signingInput := []byte(header + "." + payload)
	sig, err := signJWS(signingInput, "HS256", key)
	if err != nil {
		t.Fatalf("signJWS: %v", err)
	}
	if got := base64.RawURLEncoding.EncodeToString(sig); got != wantSigB64 {
		t.Fatalf("HS256 signature mismatch:\n got %s\nwant %s", got, wantSigB64)
	}
	// The vector should also verify.
	ok, err := verifyJWS(signingInput, sig, "HS256", key)
	if err != nil || !ok {
		t.Fatalf("HS256 verify failed: ok=%v err=%v", ok, err)
	}
}

// TestSignJWS_EdDSA_RFC8037A4 locks the Ed25519 signing path against the exact known-answer vector in RFC 8037 Appendix A.4. We build the key from the RFC's `d` seed, marshal it to PEM (the wire format the function accepts), and confirm the signature matches byte for byte.
func TestSignJWS_EdDSA_RFC8037A4(t *testing.T) {
	const (
		dB64u      = "nWGxne_9WmC6hEr0kuwsxERJxWl7MmkZcDusAxyuf2A"
		signingInp = "eyJhbGciOiJFZERTQSJ9.RXhhbXBsZSBvZiBFZDI1NTE5IHNpZ25pbmc"
		wantSigB64 = "hgyY0il_MGCjP0JzlnLWG1PPOt7-09PGcvMg3AIbQR6dWbhijcNR4ki4iylGjg5BhVsPt9g7sVvpAr_MuM0KAg"
	)
	seed, err := base64.RawURLEncoding.DecodeString(dB64u)
	if err != nil {
		t.Fatalf("decode d: %v", err)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	sig, err := signJWS([]byte(signingInp), "EdDSA", keyPEM)
	if err != nil {
		t.Fatalf("signJWS: %v", err)
	}
	if got := base64.RawURLEncoding.EncodeToString(sig); got != wantSigB64 {
		t.Fatalf("EdDSA signature mismatch:\n got %s\nwant %s", got, wantSigB64)
	}

	// Verify against the derived public key marshalled to PKIX PEM.
	pubDER, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	ok, err := verifyJWS([]byte(signingInp), sig, "EdDSA", pubPEM)
	if err != nil || !ok {
		t.Fatalf("EdDSA verify failed: ok=%v err=%v", ok, err)
	}
}

// TestSignJWS_ES256_DeterministicAndVerifies confirms ES256 signatures are the fixed 64-byte R||S form, are byte-identical across two calls (RFC 6979), and verify with the public key.
func TestSignJWS_ES256_DeterministicAndVerifies(t *testing.T) {
	priv, err := ecdsaP256KeyFromSeed([]byte("es256 jws test seed"))
	if err != nil {
		t.Fatalf("derive key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	signingInput := []byte("eyJhbGciOiJFUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9")

	sig1, err := signJWS(signingInput, "ES256", keyPEM)
	if err != nil {
		t.Fatalf("sign 1: %v", err)
	}
	sig2, err := signJWS(signingInput, "ES256", keyPEM)
	if err != nil {
		t.Fatalf("sign 2: %v", err)
	}
	if len(sig1) != 64 {
		t.Fatalf("ES256 signature must be 64 bytes (R||S), got %d", len(sig1))
	}
	if string(sig1) != string(sig2) {
		t.Fatal("ES256 signatures differ across calls; determinism broken")
	}

	pubDER, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	ok, err := verifyJWS(signingInput, sig1, "ES256", pubPEM)
	if err != nil || !ok {
		t.Fatalf("ES256 verify failed: ok=%v err=%v", ok, err)
	}

	// A tampered signing input must fail verification.
	ok, err = verifyJWS([]byte("eyJhbGciOiJFUzI1NiJ9.eyJzdWIiOiJldmUifQ"), sig1, "ES256", pubPEM)
	if err != nil {
		t.Fatalf("verify tampered: unexpected err %v", err)
	}
	if ok {
		t.Fatal("ES256 verify accepted a tampered signing input")
	}
}
