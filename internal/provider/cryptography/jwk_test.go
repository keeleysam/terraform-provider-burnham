package cryptography

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- runners ----------------------------------------------------------------

func runJWKEncode(t *testing.T, pemStr string, opts ...attr.Value) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &JWKEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.StringValue(pemStr),
		variadicDynamic(opts...),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func runJWKDecode(t *testing.T, jwk attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &JWKDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(jwk),
		variadicDynamic(),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

func runJWKThumbprint(t *testing.T, key attr.Value, hash ...string) (string, *function.FuncError) {
	t.Helper()
	f := &JWKThumbprintFunction{}
	elems := make([]attr.Value, len(hash))
	tps := make([]attr.Type, len(hash))
	for i, h := range hash {
		elems[i] = types.StringValue(h)
		tps[i] = types.StringType
	}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(key),
		types.TupleValueMust(tps, elems),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

// --- RFC 7638 thumbprint vector --------------------------------------------

func TestJWKThumbprint_RFC7638(t *testing.T) {
	// RFC 7638 section 3.1 worked example: RSA JWK -> SHA-256 thumbprint.
	const n = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
	jwkObj := mustObj(t, map[string]interface{}{
		"kty": "RSA",
		"n":   n,
		"e":   "AQAB",
		"alg": "RS256",
		"kid": "2011-04-29",
	})
	got, ferr := runJWKThumbprint(t, jwkObj)
	if ferr != nil {
		t.Fatalf("thumbprint: %v", ferr)
	}
	const want = "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"
	if got != want {
		t.Fatalf("RFC 7638 thumbprint mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestJWKThumbprint_AcceptsPEM(t *testing.T) {
	_, pubPEM := ecKeyPEMs(t, "thumbprint pem seed")
	got, ferr := runJWKThumbprint(t, types.StringValue(pubPEM))
	if ferr != nil {
		t.Fatalf("thumbprint from PEM: %v", ferr)
	}
	if got == "" {
		t.Fatal("empty thumbprint")
	}
	// Deterministic: same key -> same thumbprint.
	got2, _ := runJWKThumbprint(t, types.StringValue(pubPEM))
	if got != got2 {
		t.Fatal("thumbprint not deterministic")
	}
}

// --- encode / decode round-trips -------------------------------------------

func TestJWKEncodeDecode_RoundTrip_EC(t *testing.T) {
	priv, _ := ecdsaP256KeyFromSeed([]byte("jwk ec roundtrip"))
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	jwk, ferr := runJWKEncode(t, privPEM)
	if ferr != nil {
		t.Fatalf("encode: %v", ferr)
	}
	backPEM, ferr := runJWKDecode(t, jwk.UnderlyingValue())
	if ferr != nil {
		t.Fatalf("decode: %v", ferr)
	}
	block, _ := pem.Decode([]byte(backPEM))
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	ec, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatalf("expected *ecdsa.PrivateKey, got %T", k)
	}
	if ec.D.Cmp(priv.D) != 0 {
		t.Fatal("EC private scalar did not round-trip")
	}
}

func TestJWKEncodeDecode_RoundTrip_Ed25519(t *testing.T) {
	priv, _ := ed25519KeyFromSeed([]byte("jwk ed roundtrip"))
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	jwk, ferr := runJWKEncode(t, privPEM)
	if ferr != nil {
		t.Fatalf("encode: %v", ferr)
	}
	backPEM, ferr := runJWKDecode(t, jwk.UnderlyingValue())
	if ferr != nil {
		t.Fatalf("decode: %v", ferr)
	}
	block, _ := pem.Decode([]byte(backPEM))
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	ed, ok := k.(ed25519.PrivateKey)
	if !ok {
		t.Fatalf("expected ed25519.PrivateKey, got %T", k)
	}
	if !ed.Equal(priv) {
		t.Fatal("Ed25519 key did not round-trip")
	}
}

func TestJWKEncodeDecode_RoundTrip_RSA_Public(t *testing.T) {
	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa: %v", err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&rk.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	jwk, ferr := runJWKEncode(t, pubPEM, types.ObjectValueMust(
		map[string]attr.Type{"kid": types.StringType, "use": types.StringType},
		map[string]attr.Value{"kid": types.StringValue("k1"), "use": types.StringValue("sig")},
	))
	if ferr != nil {
		t.Fatalf("encode: %v", ferr)
	}
	// kid / use present in the JWK object.
	obj := jwk.UnderlyingValue().(types.Object)
	if obj.Attributes()["kid"].(types.String).ValueString() != "k1" {
		t.Fatal("kid option not applied")
	}

	backPEM, ferr := runJWKDecode(t, jwk.UnderlyingValue())
	if ferr != nil {
		t.Fatalf("decode: %v", ferr)
	}
	block, _ := pem.Decode([]byte(backPEM))
	if block.Type != "PUBLIC KEY" {
		t.Fatalf("expected PUBLIC KEY block, got %q", block.Type)
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	rpub, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("expected *rsa.PublicKey, got %T", pub)
	}
	if rpub.N.Cmp(rk.N) != 0 || rpub.E != rk.E {
		t.Fatal("RSA public key did not round-trip")
	}
}

// --- jwks -------------------------------------------------------------------

func TestJWKS_AssemblesSet(t *testing.T) {
	_, ecPub := ecKeyPEMs(t, "jwks ec")
	_, edPub := edKeyPEMs(t, "jwks ed")

	// Mix a PEM string and a JWK object.
	edJWK, ferr := runJWKEncode(t, edPub)
	if ferr != nil {
		t.Fatalf("encode ed: %v", ferr)
	}
	tuple := types.TupleValueMust(
		[]attr.Type{types.DynamicType, types.DynamicType},
		[]attr.Value{
			types.DynamicValue(types.StringValue(ecPub)),
			types.DynamicValue(edJWK.UnderlyingValue()),
		},
	)

	f := &JWKSFunction{}
	args := function.NewArgumentsData([]attr.Value{types.DynamicValue(tuple)})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("jwks: %v", resp.Error)
	}
	obj := resp.Result.Value().(types.Dynamic).UnderlyingValue().(types.Object)
	set := obj.Attributes()["keys"].(types.Tuple)
	if len(set.Elements()) != 2 {
		t.Fatalf("expected 2 keys in set, got %d", len(set.Elements()))
	}
	// Each element is a JWK object with a kty.
	for i, e := range set.Elements() {
		ko := e.(types.Object)
		if _, ok := ko.Attributes()["kty"]; !ok {
			t.Fatalf("keys[%d] missing kty", i)
		}
	}
}

func TestJWKThumbprint_UnknownYieldsUnknown(t *testing.T) {
	f := &JWKThumbprintFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicUnknown(),
		types.TupleValueMust([]attr.Type{}, []attr.Value{}),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !resp.Result.Value().(types.String).IsUnknown() {
		t.Fatal("expected unknown result for unknown key")
	}
}
