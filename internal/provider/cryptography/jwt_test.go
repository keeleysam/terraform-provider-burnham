package cryptography

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- test helpers -----------------------------------------------------------

func mustObj(t *testing.T, m map[string]interface{}) attr.Value {
	t.Helper()
	v, err := goJSONToTerraform(m)
	if err != nil {
		t.Fatalf("build object: %v", err)
	}
	return v
}

func variadicDynamic(opts ...attr.Value) attr.Value {
	elems := make([]attr.Value, len(opts))
	tps := make([]attr.Type, len(opts))
	for i, o := range opts {
		elems[i] = types.DynamicValue(o)
		tps[i] = types.DynamicType
	}
	return types.TupleValueMust(tps, elems)
}

func runJWTSign(t *testing.T, claims attr.Value, alg, key string, opts ...attr.Value) (string, *function.FuncError) {
	t.Helper()
	f := &JWTSignFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicValue(claims),
		types.StringValue(alg),
		types.StringValue(key),
		variadicDynamic(opts...),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	return resp.Result.Value().(types.String).ValueString(), nil
}

func runJWTVerify(t *testing.T, token, key string, opts ...attr.Value) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &JWTVerifyFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.StringValue(token),
		types.StringValue(key),
		variadicDynamic(opts...),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func edKeyPEMs(t *testing.T, seedStr string) (privPEM, pubPEM string) {
	t.Helper()
	priv, err := ed25519KeyFromSeed([]byte(seedStr))
	if err != nil {
		t.Fatalf("derive ed25519: %v", err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pubDER, _ := x509.MarshalPKIXPublicKey(priv.Public())
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privPEM, pubPEM
}

func ecKeyPEMs(t *testing.T, seedStr string) (privPEM, pubPEM string) {
	t.Helper()
	priv, err := ecdsaP256KeyFromSeed([]byte(seedStr))
	if err != nil {
		t.Fatalf("derive ecdsa: %v", err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pubDER, _ := x509.MarshalPKIXPublicKey(priv.Public())
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privPEM, pubPEM
}

// --- round-trip and determinism --------------------------------------------

func TestJWTSignDecode_RoundTrip_HS256(t *testing.T) {
	claims := mustObj(t, map[string]interface{}{
		"sub":  "1234567890",
		"name": "Ada Lovelace",
		"exp":  json.Number("1300819380"),
	})
	token, ferr := runJWTSign(t, claims, "HS256", "topsecret")
	if ferr != nil {
		t.Fatalf("sign: %v", ferr)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("token should have 3 segments: %q", token)
	}

	// Decode and confirm payload round-trips.
	f := &JWTDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(token)})
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("decode: %v", resp.Error)
	}
	obj := resp.Result.Value().(types.Dynamic).UnderlyingValue().(types.Object)
	payload := obj.Attributes()["payload"].(types.Object)
	if got := payload.Attributes()["name"].(types.String).ValueString(); got != "Ada Lovelace" {
		t.Fatalf("payload name round-trip: got %q", got)
	}
	header := obj.Attributes()["header"].(types.Object)
	if got := header.Attributes()["alg"].(types.String).ValueString(); got != "HS256" {
		t.Fatalf("header alg: got %q", got)
	}
}

func TestJWTSign_Deterministic_ES256(t *testing.T) {
	priv, _ := ecKeyPEMs(t, "es256 determinism seed")
	claims := mustObj(t, map[string]interface{}{"sub": "alice"})
	t1, ferr := runJWTSign(t, claims, "ES256", priv)
	if ferr != nil {
		t.Fatalf("sign 1: %v", ferr)
	}
	t2, ferr := runJWTSign(t, claims, "ES256", priv)
	if ferr != nil {
		t.Fatalf("sign 2: %v", ferr)
	}
	if t1 != t2 {
		t.Fatal("ES256 jwt_sign is not deterministic")
	}
}

func TestJWTSign_OptionsHeaderMerge(t *testing.T) {
	claims := mustObj(t, map[string]interface{}{"sub": "x"})
	opts := types.ObjectValueMust(
		map[string]attr.Type{"kid": types.StringType, "typ": types.StringType},
		map[string]attr.Value{"kid": types.StringValue("2025-06"), "typ": types.StringValue("JWT")},
	)
	token, ferr := runJWTSign(t, claims, "HS256", "secret", opts)
	if ferr != nil {
		t.Fatalf("sign: %v", ferr)
	}
	headerB64 := strings.Split(token, ".")[0]
	raw, _ := base64.RawURLEncoding.DecodeString(headerB64)
	var hm map[string]interface{}
	if err := json.Unmarshal(raw, &hm); err != nil {
		t.Fatalf("header json: %v", err)
	}
	if hm["kid"] != "2025-06" || hm["typ"] != "JWT" || hm["alg"] != "HS256" {
		t.Fatalf("header merge wrong: %v", hm)
	}
}

// --- verify ----------------------------------------------------------------

func verifyValid(t *testing.T, v types.Dynamic) bool {
	t.Helper()
	obj := v.UnderlyingValue().(types.Object)
	return obj.Attributes()["valid"].(types.Bool).ValueBool()
}

func TestJWTVerify_HS256_ValidAndTampered(t *testing.T) {
	claims := mustObj(t, map[string]interface{}{"sub": "bob"})
	token, ferr := runJWTSign(t, claims, "HS256", "shared-secret")
	if ferr != nil {
		t.Fatalf("sign: %v", ferr)
	}
	res, ferr := runJWTVerify(t, token, "shared-secret")
	if ferr != nil {
		t.Fatalf("verify: %v", ferr)
	}
	if !verifyValid(t, res) {
		t.Fatal("valid token reported invalid")
	}
	// Wrong secret.
	res, ferr = runJWTVerify(t, token, "wrong-secret")
	if ferr != nil {
		t.Fatalf("verify wrong: %v", ferr)
	}
	if verifyValid(t, res) {
		t.Fatal("wrong secret reported valid")
	}
	// Tamper the payload segment.
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"eve"}`)) + "." + parts[2]
	res, ferr = runJWTVerify(t, tampered, "shared-secret")
	if ferr != nil {
		t.Fatalf("verify tampered: %v", ferr)
	}
	if verifyValid(t, res) {
		t.Fatal("tampered token reported valid")
	}
}

func TestJWTVerify_ES256_And_EdDSA(t *testing.T) {
	ecPriv, ecPub := ecKeyPEMs(t, "verify es256 seed")
	claims := mustObj(t, map[string]interface{}{"sub": "carol"})
	token, ferr := runJWTSign(t, claims, "ES256", ecPriv)
	if ferr != nil {
		t.Fatalf("es256 sign: %v", ferr)
	}
	res, ferr := runJWTVerify(t, token, ecPub)
	if ferr != nil || !verifyValid(t, res) {
		t.Fatalf("es256 verify failed: %v valid=%v", ferr, verifyValid(t, res))
	}

	edPriv, edPub := edKeyPEMs(t, "verify eddsa seed")
	token, ferr = runJWTSign(t, claims, "EdDSA", edPriv)
	if ferr != nil {
		t.Fatalf("eddsa sign: %v", ferr)
	}
	res, ferr = runJWTVerify(t, token, edPub)
	if ferr != nil || !verifyValid(t, res) {
		t.Fatalf("eddsa verify failed: %v valid=%v", ferr, verifyValid(t, res))
	}
}

func TestJWTVerify_TimeClaims(t *testing.T) {
	claims := mustObj(t, map[string]interface{}{
		"sub": "dave",
		"exp": json.Number("1000"),
		"nbf": json.Number("100"),
	})
	token, ferr := runJWTSign(t, claims, "HS256", "s")
	if ferr != nil {
		t.Fatalf("sign: %v", ferr)
	}

	nowOpt := func(n string) attr.Value {
		return types.ObjectValueMust(
			map[string]attr.Type{"now": types.NumberType},
			map[string]attr.Value{"now": numberFromString(n)},
		)
	}

	// Within window (nbf <= now < exp).
	res, _ := runJWTVerify(t, token, "s", nowOpt("500"))
	if !verifyValid(t, res) {
		t.Fatal("token valid at now=500 reported invalid")
	}
	// Expired (now >= exp).
	res, _ = runJWTVerify(t, token, "s", nowOpt("2000"))
	if verifyValid(t, res) {
		t.Fatal("expired token reported valid")
	}
	// Not yet valid (now < nbf).
	res, _ = runJWTVerify(t, token, "s", nowOpt("50"))
	if verifyValid(t, res) {
		t.Fatal("not-yet-valid token reported valid")
	}
	// RFC 3339 string form for now, within window.
	rfcOpt := types.ObjectValueMust(
		map[string]attr.Type{"now": types.StringType},
		map[string]attr.Value{"now": types.StringValue("1970-01-01T00:08:20Z")}, // unix 500
	)
	res, _ = runJWTVerify(t, token, "s", rfcOpt)
	if !verifyValid(t, res) {
		t.Fatal("RFC3339 now within window reported invalid")
	}
}

func TestJWTVerify_AlgorithmPin(t *testing.T) {
	claims := mustObj(t, map[string]interface{}{"sub": "erin"})
	token, _ := runJWTSign(t, claims, "HS256", "s")
	pin := types.ObjectValueMust(
		map[string]attr.Type{"algorithm": types.StringType},
		map[string]attr.Value{"algorithm": types.StringValue("HS512")},
	)
	res, ferr := runJWTVerify(t, token, "s", pin)
	if ferr != nil {
		t.Fatalf("verify: %v", ferr)
	}
	if verifyValid(t, res) {
		t.Fatal("algorithm pin mismatch should report invalid")
	}
}

func TestJWTVerify_MalformedIsError(t *testing.T) {
	if _, ferr := runJWTVerify(t, "not.a.valid.token", "s"); ferr == nil {
		t.Fatal("expected error for 4-segment token")
	}
	if _, ferr := runJWTVerify(t, "onlyonesegment", "s"); ferr == nil {
		t.Fatal("expected error for 1-segment token")
	}
}

// --- contract: unknown / null ----------------------------------------------

func TestJWTSign_UnknownClaimsYieldsUnknown(t *testing.T) {
	f := &JWTSignFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.DynamicUnknown(),
		types.StringValue("HS256"),
		types.StringValue("s"),
		variadicDynamic(),
	})
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), function.RunRequest{Arguments: args}, resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !resp.Result.Value().(types.String).IsUnknown() {
		t.Fatal("expected unknown string result for unknown claims")
	}
}

func TestJWTSign_UnsupportedAlgorithm(t *testing.T) {
	claims := mustObj(t, map[string]interface{}{"sub": "x"})
	if _, ferr := runJWTSign(t, claims, "PS256", "s"); ferr == nil {
		t.Fatal("expected error for unsupported algorithm PS256")
	}
}

func numberFromString(s string) types.Number {
	v, err := goJSONToTerraform(json.Number(s))
	if err != nil {
		panic(err)
	}
	return v.(types.Number)
}
